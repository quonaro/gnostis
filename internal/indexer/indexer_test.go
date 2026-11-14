package indexer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/quonaro/gnostis/internal/config"
	"github.com/quonaro/gnostis/internal/directory"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func testDir(root string) directory.Directory {
	return directory.FromConfig(config.Directory{
		Path: root,
		Name: "test",
	})
}

func TestHashContent(t *testing.T) {
	got := HashContent([]byte("hello"))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Errorf("HashContent = %q, want %q", got, want)
	}
}

func TestIndexer_Index_CollectsIndexableFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "main.go"), "package main\n")
	writeTestFile(t, filepath.Join(root, "README.md"), "# hi\n")
	writeTestFile(t, filepath.Join(root, "data.txt"), "not indexed\n") // .txt not in defaults
	writeTestFile(t, filepath.Join(root, "node_modules", "dep", "index.js"), "x")

	dir := testDir(root)
	proj := directory.NewProject("test", root)

	files, err := New().Index(context.Background(), dir, proj)
	if err != nil {
		t.Fatalf("Index: %v", err)
	}

	got := map[string]string{}
	for _, f := range files {
		got[f.RelPath] = f.Content
		if f.ProjectID != "test" {
			t.Errorf("ProjectID = %q, want test", f.ProjectID)
		}
		if f.Hash == "" {
			t.Errorf("empty hash for %s", f.RelPath)
		}
		if f.ModTime.IsZero() {
			t.Errorf("zero ModTime for %s", f.RelPath)
		}
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), got)
	}
	if got["main.go"] != "package main\n" {
		t.Errorf("main.go content = %q", got["main.go"])
	}
	if got["README.md"] != "# hi\n" {
		t.Errorf("README.md content = %q", got["README.md"])
	}
}

func TestIndexer_Index_RespectsGitignore(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, ".gitignore"), "ignored.go\nsub/ignored2.go\n")
	writeTestFile(t, filepath.Join(root, "ignored.go"), "package main\n")
	writeTestFile(t, filepath.Join(root, "kept.go"), "package main\n")
	writeTestFile(t, filepath.Join(root, "sub", "ignored2.go"), "package main\n")
	writeTestFile(t, filepath.Join(root, "sub", "kept2.go"), "package main\n")

	dir := testDir(root)
	files, err := New().Index(context.Background(), dir, directory.NewProject("test", root))
	if err != nil {
		t.Fatalf("Index: %v", err)
	}

	rel := map[string]bool{}
	for _, f := range files {
		rel[f.RelPath] = true
	}
	if rel["ignored.go"] {
		t.Error("ignored.go should be excluded by root .gitignore")
	}
	if rel["sub/ignored2.go"] || rel[filepath.Join("sub", "ignored2.go")] {
		t.Error("sub/ignored2.go should be excluded by root .gitignore")
	}
	if !rel["kept.go"] {
		t.Error("kept.go should be indexed")
	}
	if !rel[filepath.Join("sub", "kept2.go")] {
		t.Error("sub/kept2.go should be indexed")
	}
}

func TestIndexer_Index_NestedGitignore(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "sub", ".gitignore"), "*.log.go\n")
	writeTestFile(t, filepath.Join(root, "sub", "a.log.go"), "package sub\n")
	writeTestFile(t, filepath.Join(root, "sub", "b.go"), "package sub\n")

	dir := testDir(root)
	files, err := New().Index(context.Background(), dir, directory.NewProject("test", root))
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if len(files) != 1 || files[0].RelPath != filepath.Join("sub", "b.go") {
		t.Fatalf("expected only sub/b.go, got %+v", files)
	}
}

func TestIndexer_Index_ContextCancelled(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "a.go"), "package main\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New().Index(ctx, testDir(root), directory.NewProject("test", root))
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestIndexer_Index_MissingRoot(t *testing.T) {
	_, err := New().Index(
		context.Background(),
		testDir(filepath.Join(t.TempDir(), "nonexistent")),
		directory.NewProject("test", "x"),
	)
	if err == nil {
		t.Fatal("expected error for missing root")
	}
}

func TestGitIgnoreCache_ConcurrentAccess(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, ".gitignore"), "*.tmp\n")
	c := newGitIgnoreCache()

	done := make(chan bool)
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				_ = c.isIgnored(root, filepath.Join(root, "x.tmp"))
			}
			done <- true
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
}

func TestIndexer_Index_ModTimeSet(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "a.go")
	writeTestFile(t, p, "package main\n")

	files, err := New().Index(context.Background(), testDir(root), directory.NewProject("test", root))
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	info, _ := os.Stat(p)
	if !files[0].ModTime.Equal(info.ModTime()) {
		t.Errorf("ModTime = %v, want %v", files[0].ModTime, info.ModTime())
	}
}
