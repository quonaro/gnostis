package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

// Server is the HTTP server. It serves the MCP streamable HTTP endpoint.
type httpServer struct {
	mcpHandler http.Handler
	mux        *http.ServeMux
}

// New creates a new HTTP server. If mcpHandler is non-nil it is mounted at
// /mcp to serve MCP clients over the Streamable HTTP transport.
func newHTTPServer(mcpHandler http.Handler) *httpServer {
	s := &httpServer{
		mcpHandler: mcpHandler,
		mux:        http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *httpServer) registerRoutes() {
	if s.mcpHandler != nil {
		s.mux.Handle("/mcp", s.mcpHandler)
	}
}

// Start launches the HTTP server. It blocks until ctx is cancelled.
func (s *httpServer) Start(ctx context.Context, port int) error {
	addr := fmt.Sprintf(":%d", port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}

	go func() {
		<-ctx.Done()
		slog.InfoContext(ctx, "shutting down web server")
		_ = srv.Shutdown(context.WithoutCancel(ctx))
	}()

	slog.InfoContext(ctx, "starting web server", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("web server: %w", err)
	}
	return nil
}
