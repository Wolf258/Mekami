# Extender Mekami

Mekami está diseñado para ser extendido. La CLI, el daemon, el lado lectura y la capa de storage son todos agnósticos del lenguaje. Agregar soporte para un nuevo lenguaje significa implementar la interfaz `api.Frontend` y registrar el indexer.

<div class="grid cards" markdown>

-   :material-file-document: **Contrato de frontend**

    El paquete `api/v1`: cada tipo y método que tu indexer debe implementar.

    [:octicons-arrow-right-24: Leer el contrato](frontend-contract.md)

-   :material-hammer-wrench: **Escribir un frontend**

    Walkthrough para agregar un nuevo lenguaje. Usa `mekami-core-rust` como ejemplo trabajado.

    [:octicons-arrow-right-24: Escribir un frontend](writing-a-frontend.md)

-   :material-cog: **El mecanismo `all_gen`**

    Cómo funciona el flujo de blank imports dev vs producción y cuándo regenerar.

    [:octicons-arrow-right-24: Mecanismo all_gen](all-gen.md)

</div>
