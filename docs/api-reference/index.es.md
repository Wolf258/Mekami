# Referencia de la API

La superficie pública de Mekami es chica. Hay dos mitades:

- El contrato `api/v1` que implementa cada indexer de lenguaje.
- La CLI / MCP / supervisor / daemon (un único binario Go).

La CLI es la superficie para el usuario; para el contrato `api/v1`, mirá la página [Frontend API](frontend-api.md).

<div class="grid cards" markdown>

-   :material-api: **Frontend API**

    `api.Frontend`, `ParseResult`, `Symbol`, `Ref`, `Workspace`, `ModuleInfo`, `ModuleEntry`, y el `Registry`.

    [:octicons-arrow-right-24: Frontend API](frontend-api.md)

</div>
