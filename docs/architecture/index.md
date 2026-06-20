# Architecture

Mekami's architecture is layered. A workspace (`go.work`) ties the three Go modules together for local development; the AUR PKGBUILD does the equivalent at build time.

## The big picture

```text
┌────────────────────────────────────────────────────────────┐
│  mekami (single binary)                                    │
│                                                            │
│  ┌──────────────────┐   ┌──────────────────────────────┐   │
│  │  cmd/mekami/     │   │  internal/mcp/server.go      │   │
│  │  (cobra CLI)     │   │  (MCP server on stdio)       │   │
│  └────────┬─────────┘   └──────────────┬───────────────┘   │
│           │                            │                   │
│           ▼                            ▼                   │
│  ┌──────────────────────────────────────────────────┐      │
│  │  internal/naming.Specs                          │      │
│  │  single source of truth: every command and tool │      │
│  └──────────────────────────────────────────────────┘      │
│           │                            │                   │
│           ▼                            ▼                   │
│  ┌──────────────────────────────────────────────────┐      │
│  │  internal/handlers/read.go                      │      │
│  │  shared read implementations (CLI + MCP)         │      │
│  └──────────────────────────────────────────────────┘      │
│           │                                                │
│           ▼                                                │
│  ┌──────────────────────────────────────────────────┐      │
│  │  internal/core/                                  │      │
│  │  ingest/  queries/  path/  diff/  grep/  store/  │      │
│  │  walk/    modlayout/                            │      │
│  │  frontend/all_gen/   (generated blank imports)  │      │
│  └──────────────────────────────────────────────────┘      │
│           │                                                │
│           ▼                                                │
│  ┌──────────────────────────────────────────────────┐      │
│  │  SQLite (.mekami/graph.db)                      │      │
│  └──────────────────────────────────────────────────┘      │
└────────────────────────────────────────────────────────────┘
```

## Key design points

### `internal/naming` is the single source of truth

Every CLI command and every MCP tool is declared as a `Spec` in `specs.go`. The CLI and the MCP server each walk the slice and register their side; renaming a tool or adding a flag is a one-line change. The CLI renders names as kebab-case; MCP as snake_case. Both use the same `Short` / `Long` descriptions.

This is why adding a new tool touches exactly one file.

### `internal/handlers` is shared

The read-side logic lives in `internal/handlers/read.go`. Both the CLI runner and the MCP server call the same functions; the only thing that differs is the wire format. If you fix a bug in `who_calls`, you fix it for both surfaces.

### `internal/core` is the indexing pipeline

`internal/core/` is the merged `mekami-core` indexing pipeline. It is split into:

- `store/` — SQLite store: open/close, transactions, row scanning.
- `walk/` — filesystem walker and fingerprint helper.
- `modlayout/` — `go.mod` / `go.work` resolution.
- `ingest/` — build orchestration: `build.go` (workspace discovery, parallelism, deletes), `incremental.go` (re-ingest without re-walking), `write.go` (language-agnostic `WriteParseResult`).
- `frontend/` — the language frontends. `all_gen/` holds the generated blank imports; concrete frontends (Go, Rust, …) live in their own repos.
- `queries/`, `path/`, `diff/`, `grep/` — read-side helpers.

The build pipeline resolves an `api.Frontend` once per `Build` and calls its `ParseFile` from a worker pool. The `api/v1` package is the public surface external indexers implement.

### `internal/watch` runs the daemon

`internal/watch` runs an `fsnotify` reader goroutine, debounces events through an internal coalescer, and dispatches to `BuildIncremental` for files handled by the active frontend or `Build` when a structural file is touched (Go: `go.mod` / `go.work` / `go.sum`; configurable per frontend via `Frontend.StructuralFiles()`). A `Source` abstraction lets the daemon swap `fsnotify` for a polling source on filesystems where inotify is unreliable (NFS, SMB, FUSE).

The daemon mode re-execs the same binary with hidden env vars so the same code path serves both modes.

### `internal/supervisor` owns the daemons

`internal/supervisor` is the per-user process that owns every watcher daemon. It uses Unix-socket IPC to talk to the CLI, and the inotify budget to make sure no single project monopolizes the kernel watch tables. It is itself supervised by `internal/watchdog`.

## Cross-cutting concerns

- **No CGo.** Mekami uses `modernc.org/sqlite`, a pure-Go SQLite driver. The binary is a single static artifact with no glibc/musl ABI mismatch.
- **Pure stdlib testing.** No testify, no ginkgo. `gofmt` + `go vet` for lint.
- **Pure stdlib `mekami-api`.** The `api/v1` package has zero internal dependencies, so a third-party frontend only needs to depend on it to register itself.

See [Modules](modules.md) for a per-package tour, and [Platform support](platform.md) for the per-OS service-manager split.
