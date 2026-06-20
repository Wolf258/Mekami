# Extending Mekami

Mekami is designed to be extended. The CLI, the daemon, the read-side, and the storage layer are all language-agnostic. Adding support for a new language means implementing the `api.Frontend` interface and registering the indexer.

<div class="grid cards" markdown>

-   :material-file-document: **Frontend contract**

    The `api/v1` package: every type and method your indexer must implement.

    [:octicons-arrow-right-24: Read the contract](frontend-contract.md)

-   :material-hammer-wrench: **Writing a frontend**

    Walkthrough for adding a new language. Uses `mekami-core-rust` as a worked example.

    [:octicons-arrow-right-24: Write a frontend](writing-a-frontend.md)

-   :material-cog: **The `all_gen` mechanism**

    How the dev vs production blank-import flow works and when to regenerate.

    [:octicons-arrow-right-24: all_gen mechanism](all-gen.md)

</div>
