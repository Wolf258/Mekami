# API reference

The public surface of Mekami is small. There are two halves:

- The `api/v1` contract every language indexer implements.
- The CLI / MCP / supervisor / daemon (a single Go binary).

The CLI is the user-facing surface; for the `api/v1` contract, see the [Frontend API](frontend-api.md) page.

<div class="grid cards" markdown>

-   :material-api: **Frontend API**

    `api.Frontend`, `ParseResult`, `Symbol`, `Ref`, `Workspace`, `ModuleInfo`, `ModuleEntry`, and the `Registry`.

    [:octicons-arrow-right-24: Frontend API](frontend-api.md)

</div>
