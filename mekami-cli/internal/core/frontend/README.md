# Language frontends

The ingest pipeline speaks a small, language-agnostic interface defined in [`api/v1`](https://wolf258.github.io/mekami/api-reference/frontend-api/). A **frontend** is a self-registering package that implements `api.Frontend` and knows how to:

1. Identify the files it claims (`Extensions()`).
2. Resolve the workspace layout for the build root (`ResolveLayout()`, returning a `*api.Workspace`).
3. Enumerate every module the build root contains (`ResolveModules()`).
4. Return the canonical root module identifier (`RootModule()`).
5. Map a file to its module/package identifiers (`ResolveFile()`).
6. Parse a single file into the generic `api.ParseResult` shape (`ParseFile()`).
7. List the basenames whose edit invalidates the whole index (`StructuralFiles()`).
8. Skip language-specific file kinds (e.g. `_test.go` in Go) from the walk (`IsIndexable()`).
9. Report its lowercase language identifier (`Name()`).

**Full documentation:** <https://wolf258.github.io/mekami/extending/writing-a-frontend/>

See also:

- [Frontend contract](https://wolf258.github.io/mekami/extending/frontend-contract/)
- [The `all_gen` mechanism](https://wolf258.github.io/mekami/extending/all-gen/)

## Adding a new language

1. Create a repo at `github.com/Wolf258/mekami-core-<lang>`.
2. Init the Go module and pull in `mekami-api`.
3. Implement `api.Frontend`.
4. Self-register at `init()`.
5. Tag the first release.
6. From the mekami source tree: `mekami core install <lang>@v0.1.0`.
7. Rebuild with `./build.sh` and verify.
