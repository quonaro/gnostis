package app

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func TestNew_MountsMCPHandler(t *testing.T) {
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})
	s := newHTTPServer(h)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	s.mux.ServeHTTP(rec, req)

	if !called {
		t.Error("mcp handler was not invoked")
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestNew_NilHandlerNoRoute(t *testing.T) {
	s := newHTTPServer(nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestServer_StartAndShutdown(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := newHTTPServer(h)

	port := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() { errCh <- s.Start(ctx, port) }()

	// Wait for server to come up.
	deadline := time.Now().Add(3 * time.Second)
	var resp *http.Response
	var lastErr error
	for time.Now().Before(deadline) {
		var err error
		resp, err = http.Post("http://127.0.0.1:"+strconv.Itoa(port)+"/mcp", "application/json", nil)
		if err == nil {
			_ = resp.Body.Close()
			break
		}
		lastErr = err
		time.Sleep(20 * time.Millisecond)
	}
	if resp == nil {
		t.Fatalf("server did not start: %v", lastErr)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Start did not return after context cancel")
	}
}
