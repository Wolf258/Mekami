# Testing

How the test suite is organised, how to run it, and the conventions
contributors must follow when adding new tests.

## At a glance

- 50 `*_test.go` files, 314 test functions, no external test-framework
  dependencies (no `testify`, no `ginkgo`).
- Two levels: **unit** (default `go test`) and **integration**
  (build tag `integration`).
- Go modules in this repo:
  - `mekami-cli/` (in this workspace, primary)
  - `mekami-core/` (in this workspace, primary)
  - `mekami-api/` (external, pulled from module proxy unless you
    use the e2e workspace template)
  - `mekami-core-go/` (external, same)

## Running the suite

### Unit tests (default)

From the repo root, with the committed `go.work`:

```
go test ./mekami-core/... ./mekami-cli/...
```

This is what CI runs and what the AUR `check()` runs. It is fast
(seconds), hermetic, and exercises every package except those
gated behind the `integration` build tag.

To run a single package:

```
go test -count=1 ./mekami-cli/cmd/mekami/
go test -count=1 -run '^TestResolveLang$' ./mekami-cli/cmd/mekami/
```

To run with the race detector:

```
go test -race ./mekami-core/... ./mekami-cli/...
```

### Integration tests

Integration tests live behind the `integration` build tag. The
default `go test` does not compile them.

For most integration tests you do not need the e2e workspace — they
are pure Go and run against `mekami-core-go` via the module proxy:

```
go test -tags integration ./mekami-core/integration_test/...
go test -tags integration ./mekami-cli/internal/watch/...
```

For the full set (including the systemd service-manager round-trip
in `service_integration_test.go`), clone the two external repos as
siblings of `Mekami/` and switch to the e2e workspace:

```
git clone https://example.com/mekami-api    ../mekami-api
git clone https://example.com/mekami-core-go ../mekami-core-go
cp go.work.e2e.example go.work
go work sync
go test -tags integration ./...
rm go.work go.work.sum
```

See `go.work.e2e.README.md` for the full workflow.

## Build tags

| Tag | Files | Purpose |
|---|---|---|
| `integration` | 20 in `mekami-core/integration_test/`, 1 in `mekami-cli/internal/watch/integration_test.go`, 1 in `mekami-cli/cmd/mekami/service_integration_test.go` | End-to-end tests that need a real `mekami-core-go` parser, the file system, or a live user bus. |
| `integration && linux` | `mekami-cli/cmd/mekami/service_integration_test.go` | Service-manager round-trip; depends on a live systemd user bus, so it is Linux-only. |
| `!integration` | `mekami-core/ingest_test/{setup,stub_frontend}_test.go` | The opposite of `integration`. Wires the stub Go frontend so the unit tests can run without the real `mekami-core-go` package. |

The build-tag form is the modern `//go:build` (Go 1.17+). We do
not keep the legacy `// +build` form; the project targets Go 1.26
and there is no compatibility reason to.

## Test matrix

| Module | Unit | Integration | Notes |
|---|---|---|---|
| `mekami-core/store` | yes | — | Upsert / upsert-parent round-trips. |
| `mekami-core/queries` | yes | — | Stats query helper. |
| `mekami-core/path` | yes | — | Error-wrap table tests. |
| `mekami-core/grep` | yes | — | grep matcher. |
| `mekami-core/ingest_test` | yes (`!integration`) | — | Stub frontend, hermetic. |
| `mekami-core/integration_test` | — | yes (20) | Real `mekami-core-go`, full build graph, prune, refs, mcp polish, etc. |
| `mekami-core/scripts/dev-allgen` | yes | — | `all_gen.go` regenerator. |
| `mekami-cli/cmd/mekami` | yes | yes (`integration && linux`) | resolveLang, resolveInitLangs, mergeIndexers, runInit, runBuild, service commands. |
| `mekami-cli/internal/config` | yes | — | Default, Load, Validate, OnStartAction, ShouldLog, Indexers. |
| `mekami-cli/internal/coreinstall` | yes | — | SplitLangRef, IsValidLang, NormalizeVersion, HighestVersion, List, Gen. |
| `mekami-cli/internal/handlers` | yes | — | Read handlers (show_body, show_changes, list_package, find_symbol, who_calls, trace_calls, find_text). |
| `mekami-cli/internal/supervisor` | yes | — | supervisor state machine, watchdog, spawn, registry, ipc, inotify budget, adopt, sentinel. |
| `mekami-cli/internal/watch` | yes | yes (1) | Filter, Coalescer, Translate, poller, paths, plus a real fsnotify integration. |
| `mekami-cli/tests/internal/install` | yes | — | Black-box MCP client registration. |
| `mekami-cli/tests/cmd/mekami` | yes | — | Black-box smoke for the `mcp-test` truncation helper. |
| `mekami-core-go` | yes (2) | — | `imports_test.go` + `external_test/func_signature_test.go`. |
| `mekami-api` | — | — | No tests. |

## Conventions

These are the rules the suite follows; new tests should follow them
too.

- **Standard `testing` only.** Use `t.Errorf` / `t.Fatalf` for
  assertions. Do not introduce `testify` or `ginkgo`.
- **Subtests via `t.Run`** for groups of related cases. Use
  snake_case subtest names that read as a path (`ok`,
  `multiple_indexers_explicit_picks_requested`).
- **Table-driven when there are ≥3 similar cases.** Define a
  `cases` slice of anonymous structs (or a map when the input is
  a natural key); each case carries a `name` for the subtest.
- **Hermetic state.** Use `t.TempDir()` for filesystem state,
  `t.Setenv()` for env, `t.Cleanup()` for everything else.
  Never reach for a global `os.Setenv` / `os.Chdir` directly.
- **`t.Helper()`** at the top of every test helper that calls
  `t.Errorf` / `t.Fatalf`.
- **No `t.Parallel()`.** Tests are fast and depend on shared
  state in places (the `api.Global` registry, the supervisor
  state). Adding parallelism is a deliberate decision, not a
  default.
- **Skip, don't fail, on environment-only gaps.** Use
  `t.Skip("reason")` when the test cannot run because of a
  missing platform prerequisite (no systemd user bus, no
  `/proc`, etc.) and add a comment explaining how to enable
  it.
- **`TestMain` is rare.** Only two test files declare one:
  the stub-frontend registrar in `ingest_test/setup_test.go`
  (build tag `!integration`) and the empty
  integration-test bootstrap in `integration_test/setup_test.go`.

## Helpers and stubs

Helpers that are reused across packages live in:

- `mekami-core/testutil/helpers.go` (production package, not
  `_test.go`). Exposes `MustMkdir`, `MustWrite`,
  `WriteModuleFiles`, `OpenStoreForTest`, `QueriesStatsForTest`.
  Black-box tests import it the same way production code does.
- `mekami-cli/internal/supervisor/testhelpers_test.go` and
  `mekami-cli/internal/watch/testhelpers_test.go` for
  package-local helpers (fsnotify shim, fake daemons, stub
  IPC servers).
- `mekami-core/integration_test/bridge_test.go:buildTestGraph`
  is the canonical "build a graph from a Go source blob"
  helper used by most integration tests.

There are three stubs of `api.Frontend` in the suite:

- `mekami-core/ingest_test/stub_frontend_test.go` — full
  `go/parser`-backed stub that returns package name and
  top-level declarations only (no imports, refs, or calls).
  Registered automatically in `TestMain` under the
  `!integration` tag.
- `mekami-cli/cmd/mekami/commands_test.go:fakeFrontend` —
  minimal in-package stub for the `resolveLang` /
  `resolveInitLangs` / `runInit` tests.
- `mekami-cli/internal/coreinstall/list_test.go:testFrontend`
  — minimal stub for the `List` tests.

They are intentionally small and not consolidated — each stub
covers only the surface the tests in its package need.

## CI and packaging

- **CI** (`.github/workflows/mekami.yml`): runs
  `go test ./...` on `mekami-cli` only, on Go 1.26. No
  `-tags integration`, so the integration suite is not
  exercised in CI. The `build` job runs `./build.sh`.
- **AUR** (`.aur/mekami/PKGBUILD:check()`): runs
  `go test ./...` in both `mekami-core` and `mekami-cli`, no
  integration tag.
- **No Makefile.** `build.sh` is a developer-only build script
  and does not run tests.

## Adding a new test

1. Pick the package the test belongs to. Prefer
   `package <name>_test` (black-box) when the test exercises
   the public surface; `package <name>` (white-box) only when
   you need access to unexported state.
2. If the test needs a real `mekami-core-go` parser, a real
   filesystem watcher, or a live user bus, gate it behind
   `//go:build integration`. If it depends on Linux systemd,
   add `&& linux`.
3. Use the conventions above: `t.TempDir()`, `t.Setenv()`,
   `t.Cleanup()`, `t.Helper()`, table-driven with `t.Run`.
4. Place shared helpers in `testutil/` (for cross-package
   helpers) or `<pkg>/testhelpers_test.go` (for package-local
   ones).
5. Run the suite locally:
   ```
   go test ./...
   go test -tags integration ./...
   gofmt -l .  # must be empty
   go vet ./...
   ```
6. CI does not run the integration suite. If your change
   depends on integration tests passing, run them locally
   before opening a PR.
