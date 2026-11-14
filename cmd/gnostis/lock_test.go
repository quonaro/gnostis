package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestTryLock_AcquireAndRelease(t *testing.T) {
	dir := t.TempDir()
	l := newFileLock(dir)

	if err := l.TryLock(); err != nil {
		t.Fatalf("TryLock: %v", err)
	}

	// Lock file should contain our PID.
	data, err := os.ReadFile(filepath.Join(dir, ".lock"))
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	if len(data) == 0 {
		t.Error("lock file is empty, expected PID")
	}

	if err := l.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	// After unlock a new lock can be acquired.
	l2 := newFileLock(dir)
	if err := l2.TryLock(); err != nil {
		t.Fatalf("TryLock after unlock: %v", err)
	}
	_ = l2.Unlock()
}

func TestTryLock_ExclusiveConflict(t *testing.T) {
	dir := t.TempDir()
	l1 := newFileLock(dir)
	if err := l1.TryLock(); err != nil {
		t.Fatalf("first TryLock: %v", err)
	}
	defer func() { _ = l1.Unlock() }()

	// Same Lock instance is idempotent.
	if err := l1.TryLock(); err != nil {
		t.Errorf("second TryLock on same instance: %v", err)
	}

	// A different Lock on the same dir must fail (flock is per-fd, but a
	// second open+flock on the same file conflicts).
	l2 := newFileLock(dir)
	if err := l2.TryLock(); err == nil {
		t.Error("expected lock conflict, got nil")
	}
}

func TestTryRLock_SucceedsWhenFree(t *testing.T) {
	dir := t.TempDir()
	l := newFileLock(dir)
	if err := l.TryRLock(); err != nil {
		t.Fatalf("TryRLock: %v", err)
	}
	_ = l.Unlock()
}

func TestTryRLock_ConflictsWithExclusive(t *testing.T) {
	dir := t.TempDir()
	l1 := newFileLock(dir)
	if err := l1.TryLock(); err != nil {
		t.Fatalf("TryLock: %v", err)
	}
	defer func() { _ = l1.Unlock() }()

	l2 := newFileLock(dir)
	if err := l2.TryRLock(); err == nil {
		t.Error("expected shared lock to conflict with exclusive lock")
	}
}

func TestUnlock_WithoutLock(t *testing.T) {
	l := newFileLock(t.TempDir())
	if err := l.Unlock(); err != nil {
		t.Errorf("Unlock on unlocked lock: %v", err)
	}
}

func TestStalePID(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".lock")

	// No lock file -> 0.
	l := newFileLock(dir)
	if pid := l.StalePID(); pid != 0 {
		t.Errorf("StalePID with no file = %d, want 0", pid)
	}

	// Garbage content -> 0.
	if err := os.WriteFile(lockPath, []byte("not-a-pid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if pid := l.StalePID(); pid != 0 {
		t.Errorf("StalePID with garbage = %d, want 0", pid)
	}

	// Our own live PID -> 0.
	if err := os.WriteFile(lockPath, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	if pid := l.StalePID(); pid != 0 {
		t.Errorf("StalePID with live pid = %d, want 0", pid)
	}

	// A very large PID that cannot exist -> stale.
	if err := os.WriteFile(lockPath, []byte("99999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if pid := l.StalePID(); pid != 99999999 {
		t.Errorf("StalePID = %d, want 99999999", pid)
	}
}
