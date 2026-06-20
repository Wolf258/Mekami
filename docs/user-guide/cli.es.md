# Referencia CLI

Cada comando es un verbo de primer nivel. No hay grupos padre `query` / `watch` / `mcp` — descubrí la superficie leyendo `mekami --help` una vez.

!!! tip "Vocabulario unificado"
    Cada comando de la CLI tiene su herramienta MCP equivalente. La CLI usa kebab-case (`who-calls`, `find-text`); MCP usa snake_case (`who_calls`, `find_text`). Se declaran una vez en `internal/naming.Specs` y se renderizan en ambas superficies automáticamente.

## Flags globales

| Flag | Descripción |
| --- | --- |
| `--db /path/to/graph.db` | Sobrescribe la ruta por defecto de la base de datos (`.mekami/graph.db`). Aceptado por cada subcomando. |
| `--json` | Emite JSON legible por máquina en lugar de texto humano. Aceptado por cada comando de lectura. |

## Lifecycle

| Comando | Descripción |
| --- | --- |
| `mekami init` | Crea `.mekami/config.json` y (opcionalmente) inicia el daemon de watch. |
| `mekami serve` | Corre el servidor MCP sobre stdio. |
| `mekami build` | Construye la base de datos del grafo de código. |
| `mekami stats` | Muestra los conteos por tabla y la raíz del último build. `--json` para salida de máquina. |

### Flags de `mekami init`

| Flag | Descripción |
| --- | --- |
| `--lang <list>` | Lista separada por comas de cores de lenguaje a habilitar (por defecto: todos los cores registrados en el binario — "all-available"). Repetible: `--lang go --lang rust`. |
| `--daemon auto\|yes\|no` | Inicia el daemon de watch después de init. `auto` (por defecto) pregunta en TTY y se saltea en shells no interactivos. |
| `--yes` | Asume "no" al prompt del daemon (equivalente a auto no interactivo). |
| `--verbose` | Muestra el progreso completo de `mekami build` en lugar del resumen de una línea. |

`init` escribe `.mekami/config.json`, persiste los cores elegidos en `indexers`, corre una `mekami build` inicial (salteada cuando hay más de un core configurado y no se pasó `--lang`), y opcionalmente inicia el watcher. Los `AllowedLangs` del build vienen de los `indexers` recién escritos, así que cualquier dato en `.mekami/graph.db` cuyo `lang` ya no esté tracked se elimina.

Re-correr `init` es idempotente: sin `--lang` une los `indexers` existentes con lo que el binario registre ahora; con `--lang` la lista explícita reemplaza lo que había.

### Flags de `mekami build`

| Flag | Descripción |
| --- | --- |
| `--root <path>` | Raíz del código fuente (por defecto: cwd). |
| `--lang <lang>` | Lenguaje a ingestar (por defecto `go`; el binario viene con el frontend Go). |
| `--clean` | Borra la base existente y reconstruye desde cero. |
| `--quiet` | Suprime el progreso por archivo. |
| `--jobs <n>` | Workers de parseo (`0` = `NumCPU`). |

`.mekami/config.json` es la fuente de verdad de qué lenguajes rastrea el proyecto. Antes de cada build, la lista de `indexers` se reconcilia contra las filas en `.mekami/graph.db`: cualquier archivo cuyo `lang` ya no esté en el conjunto se elimina, con una línea de log por lenguaje eliminado:

```text
build: removing data for disabled language(s): rust (12 files, 230 symbols, 1144 refs)
```

Pasar `--lang <x>` donde `<x>` aún no está en `indexers` extiende la lista in-place y loguea el cambio:

```text
build: adding new indexer "rust" to config.json. tracking now: go, rust
```

## Lecturas del grafo

Cada herramienta MCP también es un comando de la CLI. La herramienta MCP equivalente es la forma snake_case del mismo nombre.

| Comando CLI | Herramienta MCP | Descripción |
| --- | --- | --- |
| `mekami find <q>` | `find_symbol` | Búsqueda por subcadena sobre nombres de símbolos. |
| `mekami show <qn>` | `get_symbol` | La definición de un símbolo. Usá `--body` o `--header` para acotar la salida. |
| `mekami show-body <qn>` | `show_body` | El cuerpo del código de un símbolo (líneas numeradas). |
| `mekami show-lines <path> <start> [end]` | `show_lines` | Un rango de líneas de un archivo. |
| `mekami who-calls <qn>` | `who_calls` | Referencias entrantes (callers, type uses, lecturas de valor, embeds, imports). |
| `mekami what-calls <qn>` | `what_calls` | Referencias salientes distintas. |
| `mekami list-file <path>` | `list_file` | Símbolos de primer nivel en un archivo. |
| `mekami trace <from> <to>` | `trace_calls` | El camino de llamada más corto entre dos símbolos. |
| `mekami list-files [prefix]` | `list_files` | Árbol de archivos del proyecto. |
| `mekami list-package <import>` | `list_package` | Todos los símbolos de un paquete. |
| `mekami list-package-symbols <import>` | `list_package_symbols` | Símbolos de primer nivel en un paquete (JSON). |
| `mekami list-importers <import>` | `list_importers` | Paquetes que importan el dado. |
| `mekami list-modules` | `list_modules` | Módulos indexados. |
| `mekami show-modules` | `show_modules` | Resumen de paquetes por módulo. |
| `mekami show-changes` | `show_changes` | Archivos agregados/modificados/eliminados desde el último build. |
| `mekami find-text <pattern>` | `find_text` | Búsqueda regex server-side en los archivos fuente. |
| `mekami index-status` | `index_status` | Snapshot del índice (`last_root`, `last_build_at`, conteos). |

Todos los comandos de lectura aceptan `--json` para emitir JSON legible por máquina a stdout (código de salida no cero en un error real; `0` con resultado vacío en una query sin hits).

## Controles del daemon

| Comando | Descripción |
| --- | --- |
| `mekami start` | Lanza un daemon de watch para el proyecto actual (idempotente). |
| `mekami stop` | Para el daemon del proyecto actual. |
| `mekami status` | PID, uptime, contadores de batch, source del daemon. Usá `--json`. |
| `mekami restart` | Stop + start. |
| `mekami reload` | Re-lee `.mekami/config.json`; los cambios hot-only se empujan, los cold disparan un restart. |
| `mekami logs` | Tail del log del daemon. |
| `mekami service install` | Registra el supervisor como servicio de sistema (systemd --user en Linux, LaunchAgent en macOS). |
| `mekami service uninstall` | Desarma el servicio. |
| `mekami service status` | Muestra si el supervisor está registrado, habilitado y activo. |

Mirá [Modo watch](watch-mode.md) para la historia completa de supervisor / watchdog / adopción de huérfanos.

## Integración MCP

| Comando | Descripción |
| --- | --- |
| `mekami mcp install` | Registra el servidor MCP de mekami en el cliente host (OpenCode hoy). |
| `mekami mcp uninstall` | Elimina la entrada. |
| `mekami mcp test` | Lanza el servidor como subproceso y llama un muestreo de herramientas (smoke test). |

`mcp install` acepta:

| Flag | Descripción |
| --- | --- |
| `--binary /abs/path/mekami` | Pinea la entrada a un binario específico (útil para builds de dev). |
| `--name <otro>` | Registra bajo otro nombre de servidor. |
| `--disable` | Registra con `enabled: false`. |
| `--env KEY=VALUE` | Inyecta una variable de entorno. Repetible. |

## Core (indexers de lenguaje)

| Comando | Descripción |
| --- | --- |
| `mekami core install <lang>[@<version>]` | Registra un indexer de lenguaje para este proyecto. |
| `mekami core list` | Lista los cores configurados y cargados. |
| `mekami core uninstall <lang>` | Elimina un indexer de lenguaje del proyecto. |
| `mekami core status` | Muestra cores configurados vs cargados con un resumen missing/loaded. |

`core install` resuelve la versión vía el proxy de módulos de Go (`go list -m -versions`), escribe la entrada en `indexers` de `.mekami/config.json`, y regenera `mekami-cli/internal/core/frontend/all_gen/all_gen.go` con un blank import nuevo. Mirá [El mecanismo `all_gen`](../extending/all-gen.md) para la historia completa dev-vs-prod.

## Comandos ocultos

Algunos comandos están deliberadamente ocultos de `--help` porque son internos:

| Comando | Descripción |
| --- | --- |
| `supervise _run` | Interno: el entrypoint del proceso supervisor. |
| `supervise _watchdog` | Interno: el watchdog del supervisor. |
| `serve <flags...>` | Interno: el comando `start` re-exec el mismo binario con env vars ocultos. |
