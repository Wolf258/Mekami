---
hide:
  - navigation
  - toc
---

# Mekami

Un grafo de código Go respaldado por SQLite, para humanos y agentes LLM, expuesto sobre el [Model Context Protocol](https://modelcontextprotocol.io).

Mekami recorre un proyecto Go, parsea cada archivo con `go/parser` y persiste símbolos, definiciones, firmas y aristas de referencia en una única base SQLite. Corre como servidor MCP para que un agente (Claude, OpenCode, etc.) pueda hacer preguntas estructurales — *¿quién llama a `X`? ¿dónde está definido `X`? ¿cuál es el camino de llamada entre `A` y `B`?* — en lugar de grepear el árbol de fuentes. El mismo grafo se puede consultar desde la shell: cada herramienta MCP también es un comando de primer nivel de `mekami`.

Mekami **no** es un motor de búsqueda de código. Indexa únicamente nombres de símbolos y aristas de referencia; no indexa texto crudo. Para buscar subcadenas dentro de cuerpos de funciones, comentarios, strings de log o cualquier texto arbitrario, usá `mekami find-text` (o la herramienta MCP `find_text`) o la herramienta de lectura de tu editor.

## De un vistazo

- **Indexado incremental** — los archivos se hashean con `sha256`; los archivos sin cambios se saltean en el rebuild.
- **Ingesta en paralelo** — el parseo corre en `runtime.NumCPU()` workers; las escrituras se serializan en una única transacción SQLite.
- **Workspace-aware** — detecta `go.work` e indexa cada módulo `use`d desde la raíz del workspace, o solo el módulo actual cuando se corre desde un submódulo.
- **Servidor MCP** — 17 herramientas sobre stdio: búsqueda de símbolos, callers/callees, BFS de camino de llamada, outlines de archivo/paquete/módulo, rangos de código, búsqueda regex en el sistema de archivos, snapshot del índice.
- **Vocabulario unificado** — tanto la CLI como el MCP se declaran en un único lugar (`internal/naming.Specs`). Cambiá un nombre y se cambia en ambos lados.
- **Modo watch** — `mekami start` reindexa los archivos editados in-place vía `fsnotify` (con fallback a poller en NFS/SMB/FUSE), debouncing y detección de cambios estructurales que promueve ediciones a `go.mod` / `go.work` / `go.sum` a un rebuild completo. Manejado por un supervisor por usuario que gestiona reinicios, recargas de configuración, adopción de huérfanos tras un crash, y el presupuesto global de inotify entre todos tus proyectos.
- **Go puro** — sin CGo. Binario estático único respaldado por `modernc.org/sqlite`.

## Por dónde seguir

<div class="grid cards" markdown>

-   :material-rocket-launch: **Empezando**

    ---

    Instalá Mekami desde AUR, conectalo a tu cliente MCP y hacé tu primera consulta.

    [:octicons-arrow-right-24: Instalar](getting-started/installation.md)

-   :material-console: **Referencia CLI**

    ---

    Cada comando que expone Mekami, agrupado por propósito: lifecycle, lecturas del grafo, controles del daemon, service manager, MCP, gestión de cores.

    [:octicons-arrow-right-24: Ver la CLI](user-guide/cli.md)

-   :material-cog: **Cómo funciona la indexación**

    ---

    Recorré el flujo de datos desde los archivos fuente hasta el grafo en SQLite: fingerprint, AST collector, type resolver, writer.

    [:octicons-arrow-right-24: Leer el pipeline](user-guide/how-it-works.md)

-   :material-tools: **Extender Mekami**

    ---

    Agregá un nuevo frontend de lenguaje implementando la interfaz `api.Frontend`. Walkthrough completo usando Rust como ejemplo.

    [:octicons-arrow-right-24: Escribir un frontend](extending/writing-a-frontend.md)

</div>
