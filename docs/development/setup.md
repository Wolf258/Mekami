# Setup

## Prerequisites

- **Go 1.26+** (matches the version in `mekami-cli/go.mod`).
- **git**.
- **sqlite3** CLI optional, only for poking at the `.mekami/*.db` files by hand.
- A C toolchain is **not** required — Mekami uses `modernc.org/sqlite` (pure Go) and the Go toolchain only.

## Repository layout

Mekami is split across three public repositories so each external component can be consumed, versioned, and tested independently. The indexing pipeline that used to live in a separate `mekami-core` repo is now fused into the umbrella as `mekami-cli/internal/core/`:

```text
Wolf258/mekami-api         ← api/v1/ (the Frontend interface contract)
Wolf258/Mekami             ← umbrella: mekami-cli (with internal/core) + go.work
Wolf258/mekami-core-go     ← Go language frontend
```

The `Mekami` umbrella repo contains the whole binary as a single Go module at `mekami-cli/`, with the former `mekami-core` tree living under `internal/core/`. A committed `go.work` file at the repo root points at `mekami-cli` so build commands from the root keep working. The CLI blank-imports `mekami-core-go` from the generated `all_gen.go` to register the Go frontend in `api.Global`.

`mekami-api` and `mekami-core-go` remain external repositories. They are pulled from the Go module proxy by version.

All modules are published under `github.com/Wolf258/...`. The `mekami/...` prefix is not used because the GitHub org of that name is owned by someone else.

### What lives where

- **`mekami-api`** — pure stdlib, no internal deps. Just the `api.Frontend` interface and the shared data shapes (`ParseResult`, `Symbol`, `Ref`, `Workspace`, `ModuleInfo`, `ModuleEntry`). Bumping this is a major version for every downstream consumer.
- **`mekami-core`** — language-agnostic indexing pipeline: ingest, store, queries, walker, diff, grep. Imports `mekami-api` for the contract. Does **not** know about Go, Rust, etc. directly. Its only language-specific assumption is that any frontend can answer `ResolveLayout`, `ResolveModules`, `RootModule`, `ResolveFile`, `ParseFile`.
- **`mekami-cli`** — the binary. Imports `mekami-core` and blank-imports the language cores the user has installed (`core install go` etc.).
- **`mekami-core-go`** — the Go language frontend. Implements `api.Frontend` and self-registers at `init()`. Imports `mekami-api` for the contract; does **not** import `mekami-core` (which keeps the module graph acyclic).

## Basic setup

This is what a contributor who just wants to fix a CLI bug would do. No language core or core dev setup needed.

```bash
git clone https://github.com/Wolf258/Mekami
cd Mekami
go version                      # must be 1.26+

# Test everything in the workspace (cli + core).
go test ./...

# Build the binary.
./build.sh
./mekami --version
```

The committed `go.work` at the repo root pulls in `./mekami-cli` so `go test ./mekami-cli/...` from the root covers the whole binary. No manual workspace setup is needed for the common case.

`./build.sh` runs the dev-allgen script, regenerates `mekami-cli/internal/core/frontend/all_gen/all_gen.go` with whatever cores are currently resolvable, and produces a `mekami` binary in the repo root.

The CLI depends on `github.com/Wolf258/mekami-core`, `github.com/Wolf258/mekami-api`, and `github.com/Wolf258/mekami-core-go` (via `go.mod`). All three are fetched from the Go proxy by version. No `replace` directive is required.

## Local dev with multiple modules

If you want to develop `mekami-cli` together with local edits to either `mekami-api` or `mekami-core-go` so those take effect without publishing a tag, replace the relevant `require` in `mekami-cli/go.mod` with a `replace ... => ../<sibling>` directive, then re-run `go mod tidy`. The committed `go.work` in this repo is no longer used for that — the binary is one module now.

### Useful `go work` commands

```bash
# Add a new local module to the workspace.
go work use ../mekami-core-rust

# Show the current workspace definition.
go work edit -print

# Sync the workspace after editing go.mod files.
go work sync

# Remove a module from the workspace.
go work edit -dropreplace=../mekami-core-rust
```

## Common commands

```bash
# Run all tests across the workspace (uses the committed go.work).
go test ./...

# Test a single module.
( cd mekami-cli && go test ./... )

# Run only the matching tests, e.g. supervisor.
( cd mekami-cli && go test ./internal/supervisor/... )

# Regenerate the all_gen.go blank-import manifest.
( cd mekami-cli && go run ./internal/core/scripts/dev-allgen )

# Build the CLI binary.
./build.sh

# Rebuild after changing a core.
./build.sh && ./mekami core list
```

## Troubleshooting

### `pattern ./... matches no packages`

You're running `go test ./...` from a directory that has no `go.mod` and is not part of the workspace. Make sure you're at the repo root (where `go.work` lives) and that the file is intact. To run a single module in isolation:

```bash
( cd mekami-cli && go test ./... )
```

### Accidentally broke the workspace

The committed `go.work` lists only `./mekami-cli` and is meant to be tracked. If you edited it (or generated a `go.work.sum` against a local-only layout) and want to restore the committed content:

```bash
git checkout -- go.work
rm -f go.work.sum
```

If you are pointing the workspace at local clones of `mekami-api` or `mekami-core-go` for e2e work, prefer a `replace` directive in `mekami-cli/go.mod` to mutating `go.work`, so the change is local to your checkout and disappears with the working tree.
