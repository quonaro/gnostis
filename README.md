# Gnostis

A local "second brain" for developers. Gnostis indexes your projects with tree-sitter-aware chunking, stores embeddings locally, and exposes semantic search and code-graph tools to AI agents through the Model Context Protocol (MCP).

## What it does

- Watches configured directories and incrementally indexes changed files.
- Splits code into symbol-level chunks (functions, types, methods, classes).
- Stores embeddings locally using `chromem-go`; metadata lives in SQLite.
- Works with any OpenAI-compatible `/embeddings` endpoint (Ollama, LM Studio, OpenAI, etc.).
- Maintains a dedicated symbol index for fast exact symbol lookups.
- Builds a call graph and exposes graph analysis: architecture overview, community detection (Louvain), dead code, path tracing, similar-file detection.
- Provides a semantic memory store: agents can save, search, and read Markdown notes.
- Serves MCP over **stdio** (managed by the editor) and **Streamable HTTP** at `/mcp`.
- Multiple editor windows share a single primary instance via a stdio proxy — no extra configuration.

## MCP tools

- **Search & context:** `search`, `find_symbol`, `get_file_context`, `grep`, `list_files`, `directory_tree`, `get_recent_changes`, `find_similar`
- **Indexing:** `list_projects`, `add_project`, `edit_project`, `remove_project`, `rebuild_project`, `rebuild_index`, `reindex_files`, `get_index_status`, `get_index_job`, `check_index_coverage`, `detect_changes`
- **Graph:** `get_architecture`, `get_communities`, `graph_layout`, `trace_path`, `dead_code`
- **Memory:** `memory_list`, `memory_read`, `memory_search`, `memory_write`, `rebuild_memory`

---

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick start](#quick-start)
  - [Option 1: `go run` (no install needed)](#option-1-go-run-no-install-needed)
  - [Option 2: Pre-built binary](#option-2-pre-built-binary)
  - [Option 3: Build from source](#option-3-build-from-source)
- [Configuration](#configuration)
  - [Environment variables](#environment-variables)
  - [Example `.env` file](#example-env-file)
- [Adding projects](#adding-projects)
- [Connecting to your editor](#connecting-to-your-editor)
  - [Cursor](#cursor)
  - [Windsurf](#windsurf)
  - [Claude Desktop](#claude-desktop)
- [HTTP MCP endpoint](#http-mcp-endpoint)
- [How it works](#how-it-works)
- [Logs](#logs)
- [Development](#development)

---

## Prerequisites

### 1. Go (for `go run` and building from source)

- A recent Go toolchain with CGO enabled (`CGO_ENABLED=1`) — see `go.mod` for the required version.
- A C compiler (`gcc` on Linux, Xcode CLT on macOS).

### 2. An embeddings endpoint

Gnostis talks to any OpenAI-compatible `/embeddings` API. For local embeddings:

```bash
# Ollama
ollama pull nomic-embed-text
ollama serve
```

Or use OpenAI, OpenRouter, LocalAI, LM Studio, etc. See [Configuration](#configuration) for the endpoint settings.

---

## Quick start

Gnostis runs as a **stdio MCP server** — the MCP client (Cursor, Windsurf, Claude Desktop) manages the process lifecycle.

### Option 1: `go run` (no install needed)

Add to your MCP client config:

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "go",
      "args": ["run", "github.com/quonaro/gnostis/cmd/gnostis@latest"]
    }
  }
}
```

### Option 2: Pre-built binary

```bash
go install github.com/quonaro/gnostis/cmd/gnostis@latest
```

Then add to your MCP client config:

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "gnostis"
    }
  }
}
```

### Option 3: Build from source

```bash
git clone https://github.com/quonaro/GnostisMCP.git
cd GnostisMCP

# Build with version stamp
COMMIT=$(git rev-parse --short HEAD)
go build \
  -ldflags="-s -w -X main.version=$COMMIT -X github.com/quonaro/gnostis/internal/mcp.version=$COMMIT" \
  -o gnostis ./cmd/gnostis

# Move to PATH
sudo mv gnostis /usr/local/bin/
# or: mv gnostis ~/.local/bin/
```

Then add to your MCP client config (use the full path if not in `PATH`):

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "gnostis"
    }
  }
}
```

---

## Configuration

Gnostis is configured entirely through **environment variables**. No YAML/JSON config file is needed. See `.env.example` for a documented template — paths support `~` and `${VAR}` / `${VAR:-default}` expansion.

On first run, Gnostis creates:

- `~/.gnostis/data/` — SQLite database, vector store, memory notes.
- `~/.gnostis/gnostis.log` — log file.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `GS_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `GS_DATA_DIR` | `~/.gnostis/data` | Data directory (SQLite DB, vectors, memory notes) |
| `GS_EMBEDDINGS_URL` | `http://localhost:7997/v1` | Embeddings API endpoint (OpenAI-compatible) |
| `GS_EMBEDDINGS_MODEL` | `BAAI/bge-small-en-v1.5` | Embedding model name |
| `GS_EMBEDDINGS_API_KEY` | _(empty)_ | API key for the embeddings endpoint |
| `GS_EMBEDDINGS_BATCH_SIZE` | `32` | Max texts per embedding request |
| `GS_EMBEDDINGS_MAX_CHARS` | `8000` | Max characters of a chunk sent for embedding |
| `GS_WEB_PORT` | `7878` | Port for the HTTP MCP endpoint (`/mcp`) |

### Example `.env` file

```bash
# .env
GS_LOG_LEVEL=info
GS_EMBEDDINGS_URL=http://localhost:11434/v1
GS_EMBEDDINGS_MODEL=nomic-embed-text
GS_EMBEDDINGS_BATCH_SIZE=32
GS_WEB_PORT=7878
```

Load it before starting Gnostis (`set -a; source .env; set +a`) or pass variables via the `env` block in your MCP client config.

---

## Adding projects

Projects are stored in the SQLite database under `GS_DATA_DIR`. The simplest way to add one is via the `add_project` MCP tool from your AI editor:

```
add_project(path="/home/user/projects/myapp", name="myapp")
rebuild_project(project="myapp")
```

Project settings include `extensions`, `include`/`exclude` globs, and `max_file_size_mb`. To change them:

```
edit_project(name="myapp", exclude=["vendor/**", "**/*_test.go"])
rebuild_project(project="myapp")
```

To remove:

```
remove_project(name="myapp")
```

---

## Connecting to your editor

### Cursor

Add to `~/.cursor/mcp.json` (or `.cursor/mcp.json` in your project root):

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "gnostis",
      "env": {
        "GS_EMBEDDINGS_URL": "http://localhost:11434/v1",
        "GS_EMBEDDINGS_MODEL": "nomic-embed-text"
      }
    }
  }
}
```

### Windsurf

Add to your Windsurf MCP config (`~/.codeium/windsurf/mcp_config.json`):

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "gnostis",
      "env": {
        "GS_EMBEDDINGS_URL": "http://localhost:11434/v1",
        "GS_EMBEDDINGS_MODEL": "nomic-embed-text"
      }
    }
  }
}
```

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `~/.config/claude/claude_desktop_config.json` (Linux):

```json
{
  "mcpServers": {
    "gnostis": {
      "command": "/usr/local/bin/gnostis",
      "env": {
        "GS_EMBEDDINGS_URL": "http://localhost:11434/v1",
        "GS_EMBEDDINGS_MODEL": "nomic-embed-text"
      }
    }
  }
}
```

> **Tip:** If using `go run` instead of a pre-built binary, replace `"command": "gnostis"` with `"command": "go"` and add `"args": ["run", "github.com/quonaro/gnostis/cmd/gnostis@latest"]`.

> **Multiple editor windows:** When a second Gnostis process starts, it detects the primary instance via a file lock on the data directory and automatically runs as a stdio proxy, forwarding all MCP messages to the primary over HTTP. All editor windows share a single index — no extra configuration needed.

---

## HTTP MCP endpoint

In addition to stdio, the primary instance serves MCP over **Streamable HTTP** at:

```
http://localhost:7878/mcp
```

The port is configurable via `GS_WEB_PORT`. This endpoint is what the stdio proxies forward to, and it can also be used directly by MCP clients that support remote/HTTP transports.

---

## How it works

1. The MCP client spawns `gnostis` as a stdio subprocess (or connects to `http://localhost:7878/mcp` directly).
2. Gnostis loads config from environment variables.
3. It opens the SQLite database and vector store from `GS_DATA_DIR`.
4. Background workers index configured projects, watch for file changes, and build the call graph.
5. MCP tools serve semantic search, symbol lookup, graph analysis, and memory notes.
6. When the client closes stdin (or the process is stopped), Gnostis shuts down gracefully.

---

## Logs

Logs are written to:

- `~/.gnostis/gnostis.log` (file)
- stderr (console)

Set `GS_LOG_LEVEL=debug` for detailed embedding request logs and model activity.

---

## Development

### Prerequisites

- Go (see `go.mod`)
- [Air](https://github.com/air-verse/air) for live reload (`go install github.com/air-verse/air@latest`)
- `lota` task runner (see `lota.yaml`)

### Development workflow

```bash
# Run backend with live reload
lota dev
```

### Build

```bash
COMMIT=$(git rev-parse --short HEAD)
go build \
  -ldflags="-s -w -X main.version=$COMMIT -X github.com/quonaro/gnostis/internal/mcp.version=$COMMIT" \
  -o gnostis ./cmd/gnostis
```

### Test & lint

```bash
go build ./... && go vet ./... && go test ./...
```

### Project structure

```
cmd/gnostis/          Entry points (main.go, native.go, lock.go, log_handler.go)
internal/
  app/                Application orchestration, indexing, projects, watcher, HTTP server
  chunker/            Tree-sitter symbol extraction
  config/             Environment-based config loading
  db/                 SQLite metadata store
  directory/          Per-directory indexing rules
  embeddings/         OpenAI-compatible embeddings provider
  graph/              Call graph, layout, communities
  indexer/            File walking and filtering
  mcp/                MCP server and tool handlers
  memory/             Agent memory notes (Markdown + vector index)
  search/             Search orchestration and reranking
  simhash/            Similar-file detection
  status/             Job queue, progress, coverage, stats
  store/              chromem-go vector persistence
  symbol/             Symbol index
tests live next to the code they cover (*_test.go)
```
