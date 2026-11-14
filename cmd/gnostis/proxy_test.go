package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestForwardMessage_SetsHeaders(t *testing.T) {
	var gotAuth, gotSession, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Content-Type")
		gotSession = r.Header.Get("Mcp-Session-Id")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{}
	resp, err := forwardMessage(client, srv.URL, []byte(`{"jsonrpc":"2.0"}`), "sess-1")
	if err != nil {
		t.Fatalf("forwardMessage: %v", err)
	}
	_ = resp.Body.Close()

	if gotAuth != "application/json" {
		t.Errorf("Content-Type = %q", gotAuth)
	}
	if gotSession != "sess-1" {
		t.Errorf("Mcp-Session-Id = %q", gotSession)
	}
	if gotBody != `{"jsonrpc":"2.0"}` {
		t.Errorf("body = %q", gotBody)
	}
}

func TestForwardMessage_NoSessionHeaderWhenEmpty(t *testing.T) {
	var gotSession string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSession = r.Header.Get("Mcp-Session-Id")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := forwardMessage(&http.Client{}, srv.URL, []byte(`{}`), "")
	if err != nil {
		t.Fatalf("forwardMessage: %v", err)
	}
	_ = resp.Body.Close()
	if gotSession != "" {
		t.Errorf("unexpected session header %q", gotSession)
	}
}

func TestWriteResponse_JSONBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
	}
	var buf bytes.Buffer
	if err := writeResponse(resp, &buf); err != nil {
		t.Fatalf("writeResponse: %v", err)
	}
	if buf.String() != "{\"ok\":true}\n" {
		t.Errorf("output = %q", buf.String())
	}
}

func TestWriteResponse_AddsTrailingNewline(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("data")),
	}
	var buf bytes.Buffer
	if err := writeResponse(resp, &buf); err != nil {
		t.Fatalf("writeResponse: %v", err)
	}
	if buf.String() != "data\n" {
		t.Errorf("output = %q", buf.String())
	}
}

func TestWriteResponse_AlreadyHasNewline(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("data\n")),
	}
	var buf bytes.Buffer
	if err := writeResponse(resp, &buf); err != nil {
		t.Fatalf("writeResponse: %v", err)
	}
	if buf.String() != "data\n" {
		t.Errorf("output = %q", buf.String())
	}
}

func TestWriteResponse_AcceptedWritesNothing(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("ignored")),
	}
	var buf bytes.Buffer
	if err := writeResponse(resp, &buf); err != nil {
		t.Fatalf("writeResponse: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestWriteResponse_SSEStream(t *testing.T) {
	sse := "event: message\n" +
		"data: {\"jsonrpc\":\"2.0\",\"id\":1}\n" +
		"\n" +
		"data: [DONE]\n" +
		"data: {\"jsonrpc\":\"2.0\",\"id\":2}\n" +
		"\n"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(sse)),
	}
	var buf bytes.Buffer
	if err := writeResponse(resp, &buf); err != nil {
		t.Fatalf("writeResponse: %v", err)
	}

	want := "{\"jsonrpc\":\"2.0\",\"id\":1}\n{\"jsonrpc\":\"2.0\",\"id\":2}\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}

func TestWriteSSEEvents_SkipsNonDataLines(t *testing.T) {
	input := ": comment\n" +
		"event: ping\n" +
		"data:\n" +
		"data: [DONE]\n" +
		"data: {\"a\":1}\n"
	var buf bytes.Buffer
	if err := writeSSEEvents(strings.NewReader(input), &buf); err != nil {
		t.Fatalf("writeSSEEvents: %v", err)
	}
	if buf.String() != "{\"a\":1}\n" {
		t.Errorf("output = %q", buf.String())
	}
}
