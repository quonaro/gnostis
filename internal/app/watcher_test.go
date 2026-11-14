package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/quonaro/gnostis/internal/config"
	"github.com/quonaro/gnostis/internal/directory"
)

func testWatchDir(t *testing.T, root string) directory.Directory {
	t.Helper()
	return directory.FromConfig(config.Directory{Path: root, Name: "w"})
}

func TestIsWatchedOp(t *testing.T) {
	cases := []struct {
		op   fsnotify.Op
		want bool
	}{
		{fsnotify.Write, true},
		{fsnotify.Create, true},
		{fsnotify.Rename, true},
		{fsnotify.Remove, true},
		{fsnotify.Chmod, false},
		{fsnotify.Op(0), false},
	}
	for _, c := range cases {
		if got := isWatchedOp(c.op); got != c.want {
			t.Errorf("isWatchedOp(%v) = %v, want %v", c.op, got, c.want)
		}
	}
}

func TestWatcher_StartStop(t *testing.T) {
	root := t.TempDir()
	w := newFileWatcher([]directory.Directory{testWatchDir(t, root)}, func(path string) {})

	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if w.watcher == nil {
		t.Fatal("watcher not initialized after Start")
	}
	if err := w.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if w.watcher != nil {
		t.Error("watcher should be nil after Stop")
	}
}

func TestWatcher_StartMissingDir(t *testing.T) {
	// Start logs errors for unwatchable dirs but should not fail.
	w := newFileWatcher([]directory.Directory{
		testWatchDir(t, filepath.Join(t.TempDir(), "missing")),
	}, func(path string) {})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	_ = w.Stop()
}

func TestWatcher_NotifiesOnChange(t *testing.T) {
	root := t.TempDir()
	changed := make(chan string, 4)
	w := newFileWatcher([]directory.Directory{testWatchDir(t, root)}, func(path string) {
		changed <- path
	})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = w.Stop() }()

	target := filepath.Join(root, "file.go")
	if err := os.WriteFile(target, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case p := <-changed:
		if p != target {
			t.Errorf("changed path = %q, want %q", p, target)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for change notification")
	}
}

func TestWatcher_DebouncesMultipleWrites(t *testing.T) {
	root := t.TempDir()
	changed := make(chan string, 16)
	w := newFileWatcher([]directory.Directory{testWatchDir(t, root)}, func(path string) {
		changed <- path
	})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = w.Stop() }()

	target := filepath.Join(root, "file.go")
	// Several rapid writes should coalesce into one notification.
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(target, []byte("package main\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	select {
	case <-changed:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for debounced notification")
	}

	// After the flush, no further notifications should arrive.
	select {
	case p := <-changed:
		t.Errorf("unexpected extra notification for %q", p)
	case <-time.After(3 * time.Second):
	}
}

func TestWatcher_IgnoresChmod(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "file.go")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed := make(chan string, 4)
	w := newFileWatcher([]directory.Directory{testWatchDir(t, root)}, func(path string) {
		changed <- path
	})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = w.Stop() }()

	// Trigger the event handling path directly with a Chmod op.
	w.handleEvent(fsnotify.Event{Name: target, Op: fsnotify.Chmod})

	select {
	case p := <-changed:
		t.Errorf("chmod should not notify, got %q", p)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestWatcher_HandleEventAfterStop(t *testing.T) {
	root := t.TempDir()
	w := newFileWatcher([]directory.Directory{testWatchDir(t, root)}, func(path string) {})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := w.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// handleEvent on a stopped watcher must be a no-op (ctx is cancelled).
	w.handleEvent(fsnotify.Event{
		Name: filepath.Join(root, "x.go"),
		Op:   fsnotify.Write,
	})

	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.pending) != 0 {
		t.Error("pending should be empty after stop")
	}
}

func TestWatcher_FlushClearsPending(t *testing.T) {
	root := t.TempDir()
	var got []string
	w := newFileWatcher([]directory.Directory{testWatchDir(t, root)}, func(path string) {
		got = append(got, path)
	})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = w.Stop() }()

	w.mu.Lock()
	w.pending["/a.go"] = true
	w.pending["/b.go"] = true
	w.mu.Unlock()

	w.flush()

	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.pending) != 0 {
		t.Error("pending not cleared after flush")
	}
	if len(got) != 2 {
		t.Errorf("onChange called %d times, want 2", len(got))
	}
}
