# Configuration

Mekami reads its settings from `.mekami/config.json`. The file is optional; absent → sensible defaults.

## Schema

```json
{
  "version": 1,
  "watch": {
    "enabled": true,
    "debounce_ms": 250,
    "ignore": ["*.tmp", "*.swp", ".DS_Store"],
    "on_start": "build",
    "log": "info",
    "fallback": "auto",
    "poll_interval_s": 30,
    "log_level": "resumen",
    "self_terminate_on_orphan": ""
  },
  "build": {
    "jobs": 0
  },
  "indexers": {
    "go": "v0.1.0"
  }
}
```

## `indexers`

A map from language name to the version `core install` resolved.

```json
"indexers": {
  "go": "v0.1.0",
  "rust": ""
}
```

An empty value (`"rust": ""`) means the language was added by `mekami init` but `core install` hasn't run for it yet — the build still tracks it, and a later `mekami build --lang rust` (or `core install rust`) fills the version.

`core install <lang>[@<version>]` resolves the version via the Go module proxy (`go list -m -versions`), writes the entry to `indexers`, and regenerates `mekami-cli/internal/core/frontend/all_gen/all_gen.go` with a fresh blank import.

`mekami core list` and `core status` show the indexer set requested by the config versus what the running binary has registered. Frontends that are listed but whose blank import is missing are reported as `missing`.

## `watch` options

| Key | Default | Description |
| --- | --- | --- |
| `enabled` | `true` | Disable to make the daemon exit immediately on start (handy for one-shot index updates). |
| `debounce_ms` | `250` | Quiet window the coalescer waits after the last FS event before firing a rebuild. `0` disables debouncing. |
| `ignore` | `[]` | Basename globs dropped on top of the build walker's built-in exclusions (`.git`, `.mekami`, `vendor`, `node_modules`, `_dev`). |
| `on_start` | `"build"` | What the watcher does once before entering the event loop: `build` (full `Build`), `incremental`, or `skip`. |
| `log` | `"info"` | One of `info` (one line per batch), `debug` (per-event), or `quiet` (errors only). Applies to the foreground CLI. |
| `fallback` | `"auto"` | Event source: `auto` (fsnotify + poller fallback), `fsnotify` (force inotify), or `poll` (force poller). |
| `poll_interval_s` | `30` | Polling cadence when `fallback` is `poll` or the FS is detected as unreliable. |
| `log_level` | `"resumen"` | Daemon's persisted log: `resumen` (one line per batch) or `verbose` (per-event). Rotated at 1 MiB with three backups. |
| `self_terminate_on_orphan` | `""` | `time.ParseDuration` string (`"30s"`, `"10m"`, `"1h"`). Empty = never self-terminate. See [Watch mode](watch-mode.md#daemon-health-and-orphan-recovery). |

## `build` options

| Key | Default | Description |
| --- | --- | --- |
| `jobs` | `0` | Parse workers. `0` means `runtime.NumCPU()`. |

## Structural changes

Any edit to `go.mod`, `go.work`, or `go.sum` automatically promotes the current batch to a full rebuild — the watcher's incremental path is for ordinary Go source edits, where re-walking the tree is wasted work. If the watcher cannot find `last_root` in the DB (e.g. you ran `start` without ever building), it falls back to a one-shot full build.

The set of "structural" files is configurable per frontend via `Frontend.StructuralFiles()` (see [the frontend contract](../extending/frontend-contract.md)).

## Clean shutdown

The watcher daemon shuts down cleanly on `SIGINT` / `SIGTERM` (sent via `mekami stop`) and writes a final summary line with batch / file / error counters to its log.
