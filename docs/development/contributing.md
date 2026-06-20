# Contributing

## Code style

- `gofmt` and `go vet` are the source of truth. Run both before opening a PR.
- No external test frameworks. Standard `testing` only.
- No CGo. Use `modernc.org/sqlite` and the pure-Go standard library.
- No comments unless they explain the *why*. Code should speak for itself; reserve comments for invariants, surprising orderings, and references to upstream issues.

## The `all_gen` regeneration workflow

`mekami-cli/internal/core/frontend/all_gen/all_gen.go` is generated. Do not edit it by hand.

When the file needs to be regenerated, `./build.sh` does it for you. If you want to run the script in isolation:

```bash
( cd mekami-cli && go run ./internal/core/scripts/dev-allgen )
```

The script is idempotent: it always produces the full set, never a delta. If the diff is unexpected, check `go.work` and your module cache.

See [The `all_gen` mechanism](../extending/all-gen.md) for the full dev-vs-prod story.

## Version stamping

The version is stamped at build time via `-ldflags "-X ...install.version=..."`. Untouched builds report `dev`.

The `-ldflags` expression is inlined in two places, and they must be kept in lockstep:

- `build.sh` (manual dev builds — produces `./mekami` in the repo root)
- `.aur/mekami/PKGBUILD` (AUR from-source package)

If the install package ever moves the `version` variable, both files must be updated together.

## Pull request process

1. **Open an issue first** for any non-trivial change. Mekami is still early-stage, and a quick design conversation up front saves everyone time.
2. **Keep the PR focused.** One change, one PR. Split refactors from feature changes.
3. **Run the test suite locally** before pushing:
    ```bash
    go test ./...
    go test -tags integration ./...
    gofmt -l .
    go vet ./...
    ```
4. **Update the docs** if the change touches user-facing surface (CLI, MCP tools, configuration, watch mode). The docs live under `docs/` and are kept in sync across `en` and `es`.
5. **Do not commit secrets.** No API keys, no tokens, no `home/` paths.

## Adding a new language core

See the full walkthrough at [Writing a frontend](../extending/writing-a-frontend.md). The short version:

1. Create the repo at `github.com/Wolf258/mekami-core-<lang>`.
2. Init the Go module and pull in `mekami-api`.
3. Implement `api.Frontend`.
4. Self-register at `init()`.
5. Tag the first release.
6. From the mekami source tree: `mekami core install <lang>@v0.1.0`.
7. Rebuild and verify.

## Adding a new test

See [Testing](testing.md#adding-a-new-test) for the full checklist. The short version:

1. Pick the package. Prefer `package <name>_test` (black-box).
2. Use the `integration` build tag if the test needs a real frontend, a real watcher, or a live user bus.
3. Follow the conventions: `t.TempDir()`, `t.Setenv()`, `t.Cleanup()`, `t.Helper()`, table-driven, no `t.Parallel()`.
4. Place shared helpers in `testutil/` or `<pkg>/testhelpers_test.go`.
5. Run the suite locally before pushing.

## Reporting bugs

The `eval/` directory is the home for issue triage reports. Today it just contains a placeholder. Once a bug is reproduced, the report goes there as `eval/<date>-<short-slug>.md`.
