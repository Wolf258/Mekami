# go.work.e2e — local end-to-end workspace

This template wires `mekami-api` and `mekami-core-go` into the
local Go workspace so you can iterate on all four modules
simultaneously and have edits in the external repos take
effect without publishing tags.

## Usage

```bash
# 1. Clone the external repos as siblings of Mekami (only the
#    ones you actually want to edit locally):
#
#    ../Mekami/
#    ../mekami-api/
#    ../mekami-core-go/
#
# 2. From the Mekami repo root:
cp go.work.e2e.example go.work
go work sync

# 3. Verify the workspace resolves:
go work edit -print

# 4. Run tests, build, etc. — local edits in api/ and core-go/
#    are picked up:
go test ./...
./build.sh

# 5. When you're done, restore the committed cli+core workspace:
rm go.work go.work.sum
```

The generated `go.work` is gitignored. Adjust the sibling paths
in your copy if your layout differs from the one above.

## What runs against the workspace

- `go test -tags integration ./mekami-core/integration_test/...`
  (the ingest suite that depends on `mekami-core-go`).
- `go test -tags integration ./mekami-cli/internal/watch/...`
  (the watch end-to-end suite — fsnotify / poller → build →
  DB propagation).
- `go test -tags integration ./mekami-cli/cmd/mekami/... -run ServiceLifecycle`
  (the supervisor / service-install lifecycle; requires
  `systemd --user`).
- `./build.sh` (regenerates `all_gen.go` from whatever cores
  are resolvable, then builds the binary).

The `integration` build tag is the single switch that gates all
of the above. With the committed `go.work` (cli+core only) the
default `go test ./...` skips them. With the e2e workspace they
all run together:

```bash
go test -tags integration ./...
```

For the common case — fixing a CLI bug, reading core code —
the committed `go.work` that ships with the repo is enough;
you don't need this template.
