# 🧠 Gnostis

> A local code-intelligence MCP server — indexes your projects with tree-sitter chunking and embeddings, builds a call graph, and gives AI coding agents semantic search, graph analysis, and persistent memory tools over stdio and Streamable HTTP.

Gnostis watches your configured projects, splits code into symbol-level chunks, stores embeddings locally (`chromem-go` + SQLite), and exposes everything through [Model Context Protocol](https://modelprotocol.io) tools. No dashboard, no cloud — everything stays on your machine.

---

## ✨ Features

- 🔍 **Semantic code search** — `search`, `find_symbol`, `get_file_context`, `find_similar`
- 🌳 **Symbol-aware chunking** — tree-sitter extraction of functions, types, methods, classes
- 🕸️ **Call graph analysis** — architecture overview, Louvain communities, dead code, path tracing, force-directed layout
- 🧠 **Agent memory** — Markdown notes with semantic recall (`memory_*` tools)
- 📡 **stdio + Streamable HTTP** — served at `/mcp`; extra editor windows auto-proxy to the primary instance
- 🗄️ **Local-first storage** — SQLite for metadata, `chromem-go` for vectors, files under `~/.gnostis/data`
- 🔌 **Any OpenAI-compatible embeddings endpoint** — Ollama, LM Studio, OpenAI, etc.
- ⚙️ **Env-only config** — `GS_*` variables, `.env` support, `~`/`${VAR}` interpolation

---

## 🛠️ MCP Tools

Gnostis exposes 30 tools to AI assistants:

### Search & Context

| Tool                  | Description                                             |
| --------------------- | ------------------------------------------------------- |
| 🔍 `search`           | Semantic search across indexed code chunks              |
| 🎯 `find_symbol`      | Exact symbol lookup by name/kind                        |
| 📄 `get_file_context` | File content with surrounding symbol context            |
| 🧬 `find_similar`     | Find near-duplicate code blocks (simhash)             |
| 📁 `list_files`       | List indexed files with filters                         |
| 🌲 `directory_tree`   | Directory tree of an indexed project                    |
| 🕐 `get_recent_changes` | Recently changed files since last index               |
| 🔡 `grep`             | Regex search over indexed files                         |

### Projects & Indexing

| Tool                  | Description                                    |
| --------------------- | ---------------------------------------------- |
| 📋 `list_projects`    | List configured projects                       |
| ➕ `add_project`      | Add a project directory                        |
| ✏️ `edit_project`     | Update extensions/include/exclude rules        |
| ➖ `remove_project`   | Remove a project                               |
| 🔁 `rebuild_project`  | Full reindex of one project                    |
| 🏗️ `rebuild_index`    | Rebuild the whole index                        |
| ♻️ `reindex_files`    | Reindex specific files                         |
| 📊 `get_index_status` | Projects, chunks, provider, progress, job queue |
| ⏳ `get_index_job`    | Status of a specific indexing job              |
| 🩺 `check_index_coverage` | Verify which paths are covered by the index |
| 🕵️ `detect_changes`   | Detect changed/added/removed files             |

### Graph

| Tool                | Description                                             |
| ------------------- | ------------------------------------------------------- |
| 🏛️ `get_architecture` | High-level architecture of the call graph            |
| 🫂 `get_communities`  | Louvain communities: tightly coupled symbol groups  |
| 🕸️ `graph_layout`     | Force-directed layout (nodes, edges, positions)     |
| 🛤️ `trace_path`       | Trace call paths between two symbols                |
| 💀 `dead_code`        | Symbols with no detected callers                    |

### Memory

| Tool              | Description                                  |
| ----------------- | -------------------------------------------- |
| 📝 `memory_list`  | List memory notes                            |
| 📖 `memory_read`  | Read a memory note                           |
| 🔎 `memory_search`| Semantic search over memory notes and dialogues   |
| ✍️ `memory_write` | Create/update a memory note                  |
| 🔄 `rebuild_memory` | Rebuild the memory vector index            |

Memory notes are Markdown files under `~/.gnostis/data/memory` — agents can persist decisions, conventions, and context across sessions.

---

## 🚀 Quick Start

Gnostis runs as a **stdio MCP server** — the MCP client manages the process lifecycle.

### Option 1: `go run` (no install needed)

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

### Option 2: `go install`

```bash
go install github.com/quonaro/gnostis/cmd/gnostis@latest
```

```json
{ "mcpServers": { "gnostis": { "command": "gnostis" } } }
```

### Option 3: Build from source

```bash
git clone https://github.com/quonaro/gnostis.git && cd gnostis
go build -o gnostis ./cmd/gnostis
```

### Prerequisites

- Go 1.26+ with CGO (see `go.mod`)
- An OpenAI-compatible embeddings endpoint (e.g. `ollama serve` + `ollama pull nomic-embed-text`)

---

## ⚙️ Configuration

Env-only — no config file. Copy `.env.example` to `.env` or pass vars via the client's `env` block:

| Variable                  | Default                  | Description                     |
| ------------------------- | ------------------------ | ------------------------------- |
| `GS_LOG_LEVEL`            | `info`                   | `debug` / `info` / `warn` / `error` |
| `GS_DATA_DIR`             | `~/.gnostis/data`        | SQLite DB, vectors, memory notes |
| `GS_EMBEDDINGS_URL`       | `http://localhost:7997/v1` | Embeddings API endpoint       |
| `GS_EMBEDDINGS_MODEL`     | `BAAI/bge-small-en-v1.5` | Embedding model name            |
| `GS_EMBEDDINGS_API_KEY`   | _(empty)_                | API key if required             |
| `GS_EMBEDDINGS_BATCH_SIZE`| `32`                     | Texts per embedding request     |
| `GS_EMBEDDINGS_MAX_CHARS` | `8000`                   | Chunk truncation limit          |
| `GS_WEB_PORT`             | `7878`                   | HTTP MCP endpoint port (`/mcp`) |

### Adding projects

```
add_project(path="/home/user/projects/myapp", name="myapp")
rebuild_project(project="myapp")
```

### Connecting your editor

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

Works with Cursor, Windsurf, Claude Desktop — anything that speaks MCP stdio. The HTTP endpoint `http://localhost:7878/mcp` is also available for HTTP-capable clients.

---

## 🧱 Project Structure

```
cmd/gnostis/          Entry points, file lock, log handler, stdio proxy
internal/
  app/                Orchestration: indexing, projects, watcher, HTTP server
  chunker/            Tree-sitter symbol extraction
  config/             GS_* env loading
  db/                 SQLite metadata store
  directory/          Per-directory indexing rules
  embeddings/         OpenAI-compatible provider
  graph/              Call graph, layout, communities
  indexer/            File walking and filtering
  mcp/                MCP server + tool handlers
  memory/             Agent memory notes
  search/             Search orchestration
  simhash/            Similar-file detection
  status/             Job queue, progress, coverage, stats
  store/              chromem-go vector persistence
  symbol/             Symbol index
```

---

## 💻 Development

```bash
lota dev        # live reload (air)
go test ./...   # tests
go vet ./...    # vet
```

Pre-commit hooks (trailing whitespace, golangci-lint) are configured in `.pre-commit-config.yaml`.

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## 📄 License

MIT — see [LICENSE](LICENSE). Note: the `chromem-go` dependency is MPL-2.0; all other dependencies are MIT/BSD-style permissive licenses.
