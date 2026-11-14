package memory

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":        "hello-world",
		"  trim me  ":        "trim-me",
		"a/b\\c:d_e f":       "a-b-c-d-e-f",
		"!!!":                "note",
		"":                   "note",
		"UPPER lower123":     "upper-lower123",
		"already-kebab-case": "already-kebab-case",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugifyTruncates(t *testing.T) {
	long := ""
	for i := 0; i < 100; i++ {
		long += "word "
	}
	if got := slugify(long); len(got) > 60 {
		t.Fatalf("slugify length = %d, want <= 60", len(got))
	}
}

func TestParseMemoryMetadata(t *testing.T) {
	content := `# Chat trajectory: Fix the bug

- **Provider:** cascade
- **Source:** ` + "`session-123`" + `
- **Saved:** 2026-01-02T15:04:05Z

Body text.
`
	meta := parseMemoryMetadata(content)
	if meta.fileType != "chat" {
		t.Errorf("fileType = %q, want %q", meta.fileType, "chat")
	}
	if meta.title != "Fix the bug" {
		t.Errorf("title = %q, want %q", meta.title, "Fix the bug")
	}
	if meta.provider != "cascade" {
		t.Errorf("provider = %q, want %q", meta.provider, "cascade")
	}
	if meta.source != "session-123" {
		t.Errorf("source = %q, want %q", meta.source, "session-123")
	}
	if meta.date != "2026-01-02T15:04:05Z" {
		t.Errorf("date = %q", meta.date)
	}
}

func TestParseMemoryMetadataDefaults(t *testing.T) {
	meta := parseMemoryMetadata("no headers here")
	if meta.provider != "unknown" {
		t.Errorf("provider = %q, want %q", meta.provider, "unknown")
	}
	if meta.title != "Untitled" {
		t.Errorf("title = %q, want %q", meta.title, "Untitled")
	}
	if meta.fileType != "" {
		t.Errorf("fileType = %q, want empty", meta.fileType)
	}
}

func TestSortFilesNewestFirst(t *testing.T) {
	files := []FileInfo{
		{Path: "a", Date: "2026-01-01"},
		{Path: "b", Date: "2026-03-01"},
		{Path: "c", Date: "2026-02-01"},
	}
	sortFilesNewestFirst(files)
	want := []string{"b", "c", "a"}
	for i, w := range want {
		if files[i].Path != w {
			t.Fatalf("files[%d].Path = %q, want %q", i, files[i].Path, w)
		}
	}
}
