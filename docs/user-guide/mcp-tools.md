# MCP tools

The Mekami MCP server exposes 17 tools over stdio, derived from the same `naming.Specs` slice that drives the CLI. Tool names are snake_case; the matching CLI commands are kebab-case.

All tools return text content (JSON or formatted text) over MCP. Full descriptions are embedded in the server (the LLM reads them on every call), so the table below is a quick reference.

| Tool | Purpose |
| --- | --- |
| `find_symbol` | Substring search over symbol names. |
| `get_symbol` | A symbol's definition (formatted text, human-readable). |
| `show_body` | A symbol's source body (numbered lines). |
| `show_lines` | An arbitrary line range from a file. |
| `who_calls` | Incoming references (call, type-use, value, field, embed, import). |
| `what_calls` | Distinct outgoing references. |
| `list_file` | Top-level symbols in a file. |
| `list_package` | All symbols in a package. |
| `show_modules` | High-level summary of indexed modules and their packages. |
| `list_modules` | Indexed modules (JSON). |
| `list_package_symbols` | Top-level symbols declared in a given package. |
| `list_importers` | Packages that import a given package. |
| `list_files` | Project file tree from the indexed snapshot. |
| `trace_calls` | BFS to find a call path between two qualified names. |
| `show_changes` | Files added/modified/removed since the last `mekami build`. |
| `find_text` | Server-side regex search across source files. |
| `index_status` | Snapshot of the index (`last_root`, `last_build_at`, counts). |

## Common filters

Several tools accept filters:

- `kind` (`func`, `type`, `method`, `var`, `const`) on `find_symbol` — filters symbol kinds.
- `ref_kind` (`call`, `type-use`, `value`, `field`, `embed`, `import`) on `who_calls` — filters reference edge kinds.
- `path_prefix` on most listing tools — restricts to files whose path starts with the given prefix.

## Example session

> "Who calls `connectToServer` and what is the call path from `main` to it?"

The agent will:

1. Call `who_calls` with `qualified_name: "connectToServer"`.
2. Call `trace_calls` with `from: "main"` and `to: "connectToServer"`.
3. Synthesize an answer from the graph.

```text
call path: main -> server.Run -> runOnce -> connectToServer
  main                       cmd/example/main.go:42
  server.Run                 cmd/example/server.go:18
  runOnce                    internal/server/run.go:7
  connectToServer            internal/server/connect.go:31
```

## Smoke test

`mekami mcp test` spawns the server as a subprocess and exercises a sample of tools end-to-end against the indexed graph. Use it after any change to the server or after upgrading:

```bash
mekami build
mekami mcp test
```

The smoke runner reports each tool's success/failure and prints a summary line. Non-zero exit means at least one tool call failed.
