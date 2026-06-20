# Integration tests

End-to-end tests that require a real `mekami-core-go` parser, a real filesystem watcher, or a live user bus. Gated behind the `integration` build tag so the default `go test ./...` stays fast and hermetic.

**Full documentation:** <https://wolf258.github.io/mekami/development/testing/>

## Running locally

From `mekami-cli/`:

```bash
go test -tags integration ./internal/core/integration_test/...
go test -tags integration ./internal/watch/...
go test -tags integration ./cmd/mekami/ -run ServiceLifecycle
```

## Layout

- `setup_test.go` — bootstrap.
- `bridge_test.go` — `buildTestGraph` helper used by most tests.
- `*_test.go` — one file per scenario (refs, prune, mcp_polish, funclit, etc.).
