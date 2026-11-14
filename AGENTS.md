# Gnostis — Agent Rules

Code intelligence MCP server in Go: indexes project code (tree-sitter,
embeddings, symbols, call graph) and exposes ~28 MCP tools, plus a semantic
memory store for agent notes.

## Essentials

- **Only stdio + Streamable HTTP `/mcp`.** The dashboard frontend and WebSocket
  transport were deliberately removed — the MCP tool API is the product, not a
  UI. Do not reintroduce a dashboard or extra transports.
- **Env-only config (`GS_*`).** There is no YAML/JSON config file; `.env.example`
  documents everything. Don't add a config-file loader unprompted.
- **Storage split is intentional:** SQLite (`internal/db`) holds all metadata
  (projects, hashes, symbols, graph, stats); chromem-go holds only vectors
  (`code_chunks`, `memory_chunks` collections). Memory note *text* lives in
  `.md` files under `~/.gnostis/data/memory` — moving it into SQLite is an
  open decision, confirm before refactoring memory storage.
- **One vector store implementation.** `internal/store.Store` backs both the
  `code_chunks` and `memory_chunks` chromem collections (via `store.New` /
  `store.NewMemory`, differing only by collection + SQLite `scope`).
  `internal/memory` holds only note-domain logic (Manager, Indexer,
  ListFiles) — do not reintroduce a second Store type.
- **Verify with:** `go build ./... && go vet ./... && go test ./...`
  (`lota` wraps common tasks). `TestArchitecture_Hotspots` is flaky — rerun
  before treating a failure as real.
- **Commits:** no "Generated with Devin" / Co-Authored-By trailers.

## Code style

- **DRY.** Don't copy-paste logic between packages or files — extract a
  shared helper or generalize the existing one (parameterize by collection,
  scope, provider, etc.). Before adding a new function, check whether an
  existing one already does ~80% of the job. Duplicated ~300-line Stores
  were consolidated into one for exactly this reason; keep it that way.
- **KISS.** Prefer the smallest change that solves the problem. No new
  abstractions, config knobs, or transports unless the task requires them.

## Testing

Every package should have `*_test.go` coverage. When adding or changing code,
add tests for it — aim for behavior coverage, not line coverage.

What to cover, by area:

- **`internal/db`** — migrations are idempotent, all tables exist after
  `Open`, WAL mode is on, reader/writer pools work, reopen preserves data.
  Use `db.OpenTestDB(t)` for an in-memory DB.
- **`internal/indexer`** — file walking respects `ShouldIndex` rules,
  `.gitignore` at root and in nested dirs, context cancellation, SHA-256
  hashing. Build fixture trees with `t.TempDir()`.
- **`internal/status`** — job queue lifecycle (pending → running →
  done/failed), sequential execution, `Wait`, history pruning, `Snapshot`
  must not leak the job function; `Progress`/`State` persistence and ETA;
  per-project `Stats`; `CoverageStatus`/`CoverageChange` DTOs.
- **`cmd/gnostis/lock.go`** — exclusive/shared flock conflicts, unlock/relock,
  `StalePID` for missing/garbage/live/dead PIDs.
- **`internal/search`** — `Engine.Search` with fake `store.VectorStore` and
  `embeddings.Provider` implementations; path-prefix filtering, score
  boosting, topK truncation, embed/store error propagation; `isUnderPath`
  and `pathPrefixFromFilters` edge cases.
- **`internal/app/watcher.go`** — `isWatchedOp`, debounce coalescing, chmod
  ignored, no-op after `Stop`. Real fsnotify events need ~2s debounce waits —
  keep timeouts generous (≥5s).
- **`internal/app/server.go`** — `/mcp` route mounted only when handler
  non-nil, `Start` serves and shuts down cleanly on context cancel (grab a
  free port via `net.Listen("tcp", "127.0.0.1:0")`).
- **`cmd/gnostis`** — `parseLogLevel`, `loadConfig` env handling, stdio
  proxy helpers (`forwardMessage` headers, `writeResponse` JSON/newline/
  202 handling, `writeSSEEvents` parsing) against `httptest` servers.
- **`internal/mcp` tool handlers** — already covered; new tools need tests
  in `tools_*_test.go` following the existing harness.
- **Config (`internal/config`)** — `FromEnv` defaults/overrides/validation,
  `InterpolateEnv` `${VAR}`/`${VAR:-default}`/`~` expansion, project
  persistence round-trips.

Avoid: tests that need network access, real embedding API keys, or fixed
ports. Prefer `t.TempDir()`, `httptest`, and fakes over real services.
