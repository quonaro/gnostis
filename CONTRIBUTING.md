# Contributing to Gnostis

Thanks for your interest! Gnostis is a local-first code intelligence MCP server written in Go.

## Getting started

```bash
git clone https://github.com/quonaro/gnostis.git
cd gnostis
go build ./...
```

Requirements: Go (see `go.mod`), CGO enabled, a C compiler.

## Development workflow

```bash
lota dev          # live reload via air
```

## Before submitting

```bash
go build ./...
go vet ./...
go test ./...
```

Pre-commit hooks run automatically once installed:

```bash
pre-commit install
```

They enforce trailing-whitespace/EOF fixes and `golangci-lint --fix` (see `.pre-commit-config.yaml`).

## Guidelines

- **Keep it small.** Prefer the smallest change that solves the problem — no new abstractions, config knobs, or transports unless required.
- **DRY.** Don't copy-paste logic between packages; check whether an existing helper already does ~80% of the job.
- **Test your changes.** Every package should have `*_test.go` coverage — aim for behavior coverage. Use `t.TempDir()`, `httptest`, and fakes; avoid network access, real API keys, and fixed ports in tests.
- **No attribution trailers.** Commit messages should contain only the description of the change.
- **MCP tools need tests.** New tools go in `internal/mcp/tools_*_test.go` following the existing harness.

## Project layout

- `cmd/gnostis` — entry point, file lock, log handler, stdio proxy
- `internal/app` — orchestration: indexing, projects, watcher, HTTP server
- `internal/mcp` — MCP server and tool handlers
- `internal/{chunker,config,db,directory,embeddings,graph,indexer,memory,search,simhash,status,store,symbol}` — domain packages

## Reporting issues

Open an issue on GitHub with steps to reproduce, expected vs actual behavior, and your environment (OS, Go version, embeddings endpoint).
