package search

import (
	"context"
	"errors"
	"testing"

	chromem "github.com/philippgille/chromem-go"

	"github.com/quonaro/gnostis/internal/chunker"
	"github.com/quonaro/gnostis/internal/embeddings"
	"github.com/quonaro/gnostis/internal/store"
)

// fakeStore implements store.VectorStore returning canned results.
type fakeStore struct {
	results []chromem.Result
	err     error

	gotEmbedding []float32
	gotN         int
	gotFilters   map[string]string
}

func (f *fakeStore) AddChunks(ctx context.Context, chunks []chunker.Chunk, embs [][]float32) error {
	return nil
}
func (f *fakeStore) Query(ctx context.Context, embedding []float32, n int, filters map[string]string) ([]chromem.Result, error) {
	f.gotEmbedding = embedding
	f.gotN = n
	f.gotFilters = filters
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}
func (f *fakeStore) DeleteByPath(ctx context.Context, path string) error     { return nil }
func (f *fakeStore) DeleteByPaths(ctx context.Context, paths []string) error { return nil }
func (f *fakeStore) GetFileHash(ctx context.Context, path string) (string, error) {
	return "", nil
}
func (f *fakeStore) FileHashes() map[string]string { return nil }
func (f *fakeStore) Paths() []string               { return nil }
func (f *fakeStore) Count() int                    { return 0 }
func (f *fakeStore) CountByProject(ctx context.Context, projectID string) (int, error) {
	return 0, nil
}

var _ store.VectorStore = (*fakeStore)(nil)

// fakeProvider implements embeddings.Provider.
type fakeProvider struct {
	vectors [][]float32
	err     error
}

func (f *fakeProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.vectors, nil
}
func (f *fakeProvider) BatchSize() int    { return 32 }
func (f *fakeProvider) ModelName() string { return "fake" }

var _ embeddings.Provider = (*fakeProvider)(nil)

func chromemResult(id, path, symbol string, similarity float32) chromem.Result {
	return chromem.Result{
		ID:         id,
		Content:    "content of " + id,
		Similarity: similarity,
		Metadata: map[string]string{
			"project_id": "proj",
			"path":       path,
			"language":   "go",
			"symbol":     symbol,
			"signature":  "func " + symbol,
			"start_line": "10",
			"end_line":   "20",
		},
	}
}

func TestSearch_BasicQuery(t *testing.T) {
	fs := &fakeStore{results: []chromem.Result{
		chromemResult("a", "/proj/main.go", "main", 0.9),
		chromemResult("b", "/proj/util.go", "helper", 0.5),
	}}
	fp := &fakeProvider{vectors: [][]float32{{0.1, 0.2}}}
	e := New(fs, fp)

	res, err := e.Search(context.Background(), "find main", nil, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].ID != "a" {
		t.Errorf("top result = %q, want a (higher score)", res[0].ID)
	}
	if res[0].Path != "/proj/main.go" || res[0].Symbol != "main" {
		t.Errorf("unexpected result fields: %+v", res[0])
	}
	if res[0].StartLine != 10 || res[0].EndLine != 20 {
		t.Errorf("line numbers not parsed: %+v", res[0])
	}
	if fs.gotN != 20 { // topK*2 with topK=10
		t.Errorf("store query n = %d, want 20", fs.gotN)
	}
	if len(fs.gotEmbedding) != 2 {
		t.Errorf("embedding not passed through")
	}
}

func TestSearch_DefaultTopK(t *testing.T) {
	fs := &fakeStore{}
	fp := &fakeProvider{vectors: [][]float32{{1}}}
	e := New(fs, fp)

	_, err := e.Search(context.Background(), "q", nil, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if fs.gotN != 20 {
		t.Errorf("n = %d, want 20 (default topK=10 * 2)", fs.gotN)
	}
}

func TestSearch_PathFilter(t *testing.T) {
	fs := &fakeStore{results: []chromem.Result{
		chromemResult("in", "/proj/sub/a.go", "a", 0.9),
		chromemResult("out", "/other/b.go", "b", 0.8),
		chromemResult("edge", "/projfile/c.go", "c", 0.7), // prefix but not under path
	}}
	fp := &fakeProvider{vectors: [][]float32{{1}}}
	e := New(fs, fp)

	res, err := e.Search(context.Background(), "q",
		map[string]string{"path": "/proj", "language": "go"}, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res) != 1 || res[0].ID != "in" {
		t.Fatalf("expected only 'in' result, got %+v", res)
	}
	// "path" must be stripped from metadata filters sent to the store.
	if _, ok := fs.gotFilters["path"]; ok {
		t.Error("path filter should not be forwarded to store")
	}
	if fs.gotFilters["language"] != "go" {
		t.Errorf("language filter not forwarded: %v", fs.gotFilters)
	}
	if fs.gotN != 40 { // topK*4 when path filter present
		t.Errorf("n = %d, want 40", fs.gotN)
	}
}

func TestSearch_RootPathFilterMatchesEverything(t *testing.T) {
	fs := &fakeStore{results: []chromem.Result{
		chromemResult("a", "/anywhere/x.go", "x", 0.9),
	}}
	fp := &fakeProvider{vectors: [][]float32{{1}}}
	e := New(fs, fp)

	res, err := e.Search(context.Background(), "q",
		map[string]string{"path": "/"}, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
}

func TestSearch_EmbedError(t *testing.T) {
	fs := &fakeStore{}
	fp := &fakeProvider{err: errors.New("embed fail")}
	e := New(fs, fp)

	_, err := e.Search(context.Background(), "q", nil, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSearch_EmptyEmbedding(t *testing.T) {
	fs := &fakeStore{}
	fp := &fakeProvider{vectors: [][]float32{}}
	e := New(fs, fp)

	_, err := e.Search(context.Background(), "q", nil, 10)
	if err == nil {
		t.Fatal("expected error for empty embedding")
	}
}

func TestSearch_StoreError(t *testing.T) {
	fs := &fakeStore{err: errors.New("store fail")}
	fp := &fakeProvider{vectors: [][]float32{{1}}}
	e := New(fs, fp)

	_, err := e.Search(context.Background(), "q", nil, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSearch_ScoreBoost(t *testing.T) {
	fs := &fakeStore{results: []chromem.Result{
		chromemResult("sym", "/x/y.go", "FindHandler", 0.5),
		chromemResult("path", "/find/z.go", "zzz", 0.5),
		chromemResult("none", "/x/w.go", "www", 0.5),
	}}
	fp := &fakeProvider{vectors: [][]float32{{1}}}
	e := New(fs, fp)

	res, err := e.Search(context.Background(), "find", nil, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("expected 3 results, got %d", len(res))
	}
	// symbol match (+0.1) ranks above path match (+0.05) above none.
	if res[0].ID != "sym" || res[1].ID != "path" || res[2].ID != "none" {
		t.Errorf("unexpected ordering: %s, %s, %s", res[0].ID, res[1].ID, res[2].ID)
	}
	if res[0].Score <= res[1].Score || res[1].Score <= res[2].Score {
		t.Errorf("scores not boosted as expected: %v, %v, %v",
			res[0].Score, res[1].Score, res[2].Score)
	}
}

func TestSearch_TopKTruncation(t *testing.T) {
	results := make([]chromem.Result, 0, 15)
	for i := 0; i < 15; i++ {
		results = append(results, chromemResult(string(rune('a'+i)), "/p/f.go", "s", float32(0.9-float64(i)*0.01)))
	}
	fs := &fakeStore{results: results}
	fp := &fakeProvider{vectors: [][]float32{{1}}}
	e := New(fs, fp)

	res, err := e.Search(context.Background(), "q", nil, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(res) != 5 {
		t.Errorf("expected 5 results, got %d", len(res))
	}
}

func TestResultFromChromem(t *testing.T) {
	r := chromem.Result{
		ID:         "id1",
		Content:    "body",
		Similarity: 0.42,
		Metadata: map[string]string{
			"project_id": "p",
			"path":       "/x.go",
			"language":   "go",
			"symbol":     "sym",
			"signature":  "sig",
			"start_line": "3",
			"end_line":   "bad", // atoi must tolerate
		},
	}
	res := resultFromChromem(r)
	if res.ID != "id1" || res.Content != "body" || res.Score != 0.42 {
		t.Errorf("basic fields wrong: %+v", res)
	}
	if res.StartLine != 3 || res.EndLine != 0 {
		t.Errorf("line parsing wrong: %+v", res)
	}
}

func TestIsUnderPath(t *testing.T) {
	cases := []struct {
		path, root string
		want       bool
	}{
		{"/a/b/c.go", "/a/b", true},
		{"/a/b", "/a/b", true},
		{"/a/bc/d.go", "/a/b", false}, // prefix but not path boundary
		{"/a/b/c.go", "/a/b/c.go", true},
		{"/x.go", "/", true},
		{"/x.go", "/a", false},
		{"", "/a", false},
	}
	for _, c := range cases {
		if got := isUnderPath(c.path, c.root); got != c.want {
			t.Errorf("isUnderPath(%q, %q) = %v, want %v", c.path, c.root, got, c.want)
		}
	}
}

func TestPathPrefixFromFilters(t *testing.T) {
	prefix, rest := pathPrefixFromFilters(nil)
	if prefix != "" || rest != nil {
		t.Errorf("nil filters: got %q, %v", prefix, rest)
	}

	prefix, rest = pathPrefixFromFilters(map[string]string{"language": "go"})
	if prefix != "" || rest["language"] != "go" {
		t.Errorf("no path filter: got %q, %v", prefix, rest)
	}

	prefix, rest = pathPrefixFromFilters(map[string]string{
		"path":     "/p",
		"language": "go",
	})
	if prefix != "/p" || len(rest) != 1 || rest["language"] != "go" {
		t.Errorf("path filter: got %q, %v", prefix, rest)
	}
}
