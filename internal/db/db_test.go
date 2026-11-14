package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen_CreatesDatabaseAndMigrates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "gnostis.db")

	d, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = d.Close() }()

	if d.Writer() == nil || d.Reader() == nil {
		t.Fatal("expected non-nil reader and writer pools")
	}

	// All migrated tables should exist.
	tables := []string{
		"projects", "file_hashes", "embedding_dim", "symbols",
		"graph_nodes", "graph_edges", "simhash_entries",
		"progress_state", "project_stats",
	}
	for _, table := range tables {
		var name string
		err := d.Reader().QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing: %v", table, err)
		}
	}

	// WAL mode should be enabled.
	var journal string
	if err := d.Reader().QueryRow(`PRAGMA journal_mode`).Scan(&journal); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if journal != "wal" {
		t.Errorf("expected WAL journal mode, got %q", journal)
	}
}

func TestOpen_ReopenExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gnostis.db")

	d1, err := Open(path)
	if err != nil {
		t.Fatalf("Open first: %v", err)
	}
	if _, err := d1.Writer().Exec(
		`INSERT INTO projects (name, path) VALUES ('p1', '/tmp/p1')`,
	); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if err := d1.Close(); err != nil {
		t.Fatalf("close first: %v", err)
	}

	d2, err := Open(path)
	if err != nil {
		t.Fatalf("Open second: %v", err)
	}
	defer func() { _ = d2.Close() }()

	var name string
	if err := d2.Reader().QueryRow(`SELECT name FROM projects`).Scan(&name); err != nil {
		t.Fatalf("read project after reopen: %v", err)
	}
	if name != "p1" {
		t.Errorf("expected project p1, got %q", name)
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	// A path inside a regular file cannot be created.
	f := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if _, err := Open(filepath.Join(f, "db.sqlite")); err == nil {
		t.Fatal("expected error opening db under a file path")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	d := OpenTestDB(t)
	// Running Migrate again must not fail.
	if err := Migrate(d.Writer()); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

func TestClose(t *testing.T) {
	d := OpenTestDB(t)
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Closing twice should return an error from sql.DB, not panic.
	_ = d.Close()
}
