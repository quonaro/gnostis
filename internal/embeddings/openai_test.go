package embeddings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestEmbed_ConcurrencySerialized(t *testing.T) {
	var callCount int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2],"index":0}]}`))
	}))
	defer srv.Close()

	p := newOpenAICompatible(srv.URL, "test-model", "", 32, 8000)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := p.Embed(context.Background(), []string{"test"})
			if err != nil {
				t.Errorf("Embed failed: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if callCount != 3 {
		t.Errorf("expected 3 API calls, got %d", callCount)
	}
}

func TestEmbed_EmptyInputsReplaced(t *testing.T) {
	var gotInput []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotInput = body.Input
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1],"index":0},{"embedding":[0.2],"index":1},{"embedding":[0.3],"index":2}]}`))
	}))
	defer srv.Close()

	p := newOpenAICompatible(srv.URL, "test-model", "", 32, 8000)

	vecs, err := p.Embed(context.Background(), []string{"", "  \n\t ", "real"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vecs))
	}
	if len(gotInput) != 3 {
		t.Fatalf("expected 3 inputs, got %d", len(gotInput))
	}
	for i, in := range gotInput {
		if in == "" {
			t.Errorf("input %d is empty, want placeholder", i)
		}
	}
}

func TestEmbed_ConcurrentCallsSucceed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2],"index":0}]}`))
	}))
	defer srv.Close()

	p := newOpenAICompatible(srv.URL, "test-model", "", 32, 8000)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := p.Embed(context.Background(), []string{"test"})
			if err != nil {
				t.Errorf("Embed failed: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
}
