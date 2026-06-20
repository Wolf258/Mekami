# Frontend contract

The `api/v1` package (`github.com/Wolf258/mekami-api/api/v1`) is the public contract every language indexer implements. The package only depends on the Go standard library, so external frontends only need to depend on it to register themselves.

For the full godoc-rendered reference, see the [Frontend API reference](../api-reference/frontend-api.md).

## The `Frontend` interface

A frontend is a self-registering package that knows how to:

1. Identify the files it claims (`Extensions()`).
2. Resolve the workspace layout for the build root (`ResolveLayout()`).
3. Enumerate every module the build root contains (`ResolveModules()`).
4. Return the canonical root module identifier (`RootModule()`).
5. Map a file to its module/package identifiers (`ResolveFile()`).
6. Parse a single file into the generic `ParseResult` shape (`ParseFile()`).
7. List the basenames whose edit invalidates the whole index (`StructuralFiles()`).
8. Skip language-specific file kinds from the walk (`IsIndexable()`).
9. Report its lowercase language identifier (`Name()`).

## Data shapes

### `Workspace`

Describes a multi-module layout. Indexers without a workspace concept return `IsWorkspace: false` and an empty `WorkspaceMods`.

| Field | Description |
| --- | --- |
| `IsWorkspace` | True if the project has a workspace manifest. |
| `WorkFile` | Absolute path of the manifest (`go.work`, `Cargo.toml` workspace, …). |
| `WorkspaceDir` | Dir containing the manifest. |
| `WorkspaceMods` | Absolute dirs of every `use`d / `member` module. |
| `PrimaryModPath` | Canonical module path of the primary module. |
| `PrimaryModuleDir` | Absolute dir of the primary module. |

### `FileMeta`

Resolved package/module info for a single source file.

| Field | Description |
| --- | --- |
| `ModuleID` | Language-agnostic module identifier. |
| `PackageID` | Language-agnostic package identifier (within the module). |
| `DirRel` | Directory of the file relative to the build root. |

### `ModuleInfo`

A single module the frontend discovered in the build root. The result of `ResolveModules`.

| Field | Description |
| --- | --- |
| `Dir` | Absolute filesystem path of the module root. |
| `ModuleID` | Canonical module identifier. |

### `ParseResult`

The CPU-only output of parsing a single file. The ingest pipeline writes it as-is to the SQLite store; nothing in the result struct is language-specific.

| Field | Description |
| --- | --- |
| `RelPath` | Path of the file relative to the build root. |
| `Lang` | Stable lowercase identifier. The pipeline uses it to short-circuit re-ingest when a file's language changes. |
| `ModuleID` | The file's module identifier. |
| `PackageID` | The file's package identifier. |
| `DirRel` | Directory of the file relative to the build root. |
| `Hash` | SHA-256 hash of the file contents. |
| `Mtime` | File modification time. |
| `Size` | File size in bytes. |
| `Symbols` | Symbols declared in this file. |
| `Refs` | Reference edges originating in this file. |

### `Symbol` and `SymbolKind`

A `Symbol` is a single declaration. The `file_id` and `package_id` fields are stamped by the core after the indexer returns; the indexer must leave them zero.

| Field | Description |
| --- | --- |
| `Kind` | One of `KindFunc`, `KindMethod`, `KindType`, `KindVar`, `KindConst`, `KindImports`, `KindFuncLit`. |
| `Name` | Bare symbol name. |
| `QualifiedName` | `package.Symbol` form. |
| `StartLine` / `EndLine` | 1-based line range (inclusive). |
| `Exported` | True if the symbol is exported. |
| `Signature` | Rendered signature for display. |
| `ParentSymbol` | Pointer to the parent symbol id (set by the core, not the indexer). |

`KindImports` is a synthetic anchor for the file's import block. `KindFuncLit` is a synthetic owner for top-level `*ast.FuncLit` nodes (e.g. `&cobra.Command{RunE: func(...) error {...}}`); assigning it a unique qualified name keeps every call inside the closure visible in `who_calls` and `trace_calls`.

### `Ref` and `RefKind`

A `Ref` is a single reference edge. `FromSymbol` is the 0-based index of the originating symbol within the same `ParseResult.Symbols` slice; the core translates it to a real DB id during the write phase.

| Field | Description |
| --- | --- |
| `FromSymbol` | 0-based index into `ParseResult.Symbols`. |
| `ToQualified` | The qualified name being referenced. |
| `Kind` | One of `RefCall`, `RefTypeUse`, `RefImport`, `RefValue`. |
| `Line` | 1-based source line. |

## The registry

`api.Global` is the default registry. Indexers call `api.Register(f)` from an `init()` function; the binary's `main` blank-imports the frontend packages that need to register. Duplicate names panic at startup so a typo is caught early.

```go
func init() { api.Register(Frontend{}) }
```

## Contract guarantees

A frontend **MUST**:

- Be safe for concurrent `ParseFile` / `ResolveFile` calls (the pipeline runs N workers).
- Return a non-nil `Symbols` and `Refs` slice (empty is fine) — never `nil`. The writer indexes them directly.
- Set `ParseResult.Lang` to a stable lowercase identifier. The pipeline uses it to short-circuit re-ingest when the file's language changes (e.g. a `.go` file is renamed to `.py`).
- Set `Refs[i].FromSymbol` to the 0-based index of the originating symbol in `Symbols`. The writer translates that to a real DB id after the symbols are inserted. The Go frontend uses a 0-based index; do the same in yours.

A frontend **MAY**:

- Return an empty `Workspace` from `ResolveLayout` if the language has no workspace concept.
- Return `""` from `RootModule` if the language has no canonical root module.
- Return an empty `StructuralFiles` if every edit should be handled by the incremental path.

A frontend **SHOULD NOT**:

- Touch the database directly. Persist via `api.ParseResult` and let `ingest.WriteParseResult` handle the DB write.
- Block on I/O outside its own file. Workers run in parallel and one slow frontend would stall the pool.

## Schema compatibility

The store, queries, MCP tools and DTOs are language-agnostic. The `packages` table uses two identifier columns:

- `module_id` — a string naming the build's module (Go: `github.com/foo/bar`; Python: project name from `pyproject.toml`; Rust: crate name).
- `package_id` — a string identifying the package within the module (Go: full import path; Python: dotted module path; Rust: `crate::module`).

Frontends that have no concept of a sub-package can use the `module_id` as the `package_id` and the directory basename as the `name` column.
