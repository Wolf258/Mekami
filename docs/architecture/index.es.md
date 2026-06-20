# Arquitectura

La arquitectura de Mekami está en capas. Un workspace (`go.work`) ata los tres módulos Go juntos para desarrollo local; el PKGBUILD de AUR hace el equivalente en tiempo de compilación.

## El panorama general

```text
┌────────────────────────────────────────────────────────────┐
│  mekami (binario único)                                    │
│                                                            │
│  ┌──────────────────┐   ┌──────────────────────────────┐   │
│  │  cmd/mekami/     │   │  internal/mcp/server.go      │   │
│  │  (cobra CLI)     │   │  (servidor MCP sobre stdio)  │   │
│  └────────┬─────────┘   └──────────────┬───────────────┘   │
│           │                            │                   │
│           ▼                            ▼                   │
│  ┌──────────────────────────────────────────────────┐      │
│  │  internal/naming.Specs                          │      │
│  │  única fuente de verdad: cada comando y tool    │      │
│  └──────────────────────────────────────────────────┘      │
│           │                            │                   │
│           ▼                            ▼                   │
│  ┌──────────────────────────────────────────────────┐      │
│  │  internal/handlers/read.go                      │      │
│  │  implementaciones de lectura compartidas         │      │
│  │  (CLI + MCP)                                    │      │
│  └──────────────────────────────────────────────────┘      │
│           │                                                │
│           ▼                                                │
│  ┌──────────────────────────────────────────────────┐      │
│  │  internal/core/                                  │      │
│  │  ingest/  queries/  path/  diff/  grep/  store/  │      │
│  │  walk/    modlayout/                            │      │
│  │  frontend/all_gen/   (blank imports generados)  │      │
│  └──────────────────────────────────────────────────┘      │
│           │                                                │
│           ▼                                                │
│  ┌──────────────────────────────────────────────────┐      │
│  │  SQLite (.mekami/graph.db)                      │      │
│  └──────────────────────────────────────────────────┘      │
└────────────────────────────────────────────────────────────┘
```

## Puntos clave de diseño

### `internal/naming` es la única fuente de verdad

Cada comando de la CLI y cada herramienta MCP se declaran como un `Spec` en `specs.go`. La CLI y el servidor MCP cada uno recorre el slice y registra su lado; renombrar una herramienta o agregar un flag es un cambio de una línea. La CLI renderiza los nombres como kebab-case; MCP como snake_case. Ambos usan las mismas descripciones `Short` / `Long`.

Por eso agregar una herramienta nueva toca exactamente un archivo.

### `internal/handlers` es compartido

La lógica del lado lectura vive en `internal/handlers/read.go`. Tanto el runner de la CLI como el servidor MCP llaman a las mismas funciones; lo único que difiere es el formato del wire. Si arreglás un bug en `who_calls`, lo arreglás para ambas superficies.

### `internal/core` es el pipeline de indexación

`internal/core/` es el pipeline de indexación de `mekami-core` mergeado. Está dividido en:

- `store/` — store de SQLite: open/close, transacciones, scan de filas.
- `walk/` — walker del sistema de archivos y helper de fingerprint.
- `modlayout/` — resolución de `go.mod` / `go.work`.
- `ingest/` — orquestación de la build: `build.go` (descubrimiento del workspace, paralelismo, borrados), `incremental.go` (re-ingesta sin re-recorrer), `write.go` (`WriteParseResult` agnóstico de lenguaje).
- `frontend/` — los frontends de lenguaje. `all_gen/` tiene los blank imports generados; los frontends concretos (Go, Rust, …) viven en sus propios repos.
- `queries/`, `path/`, `diff/`, `grep/` — helpers del lado lectura.

El pipeline de build resuelve un `api.Frontend` una vez por `Build` y llama a su `ParseFile` desde un pool de workers. El paquete `api/v1` es la superficie pública que los indexers externos implementan.

### `internal/watch` corre el daemon

`internal/watch` corre una goroutine lectora de `fsnotify`, debouncea los eventos a través de un coalescer interno, y despacha a `BuildIncremental` para los archivos manejados por el frontend activo o `Build` cuando se toca un archivo estructural (Go: `go.mod` / `go.work` / `go.sum`; configurable por frontend vía `Frontend.StructuralFiles()`). Una abstracción `Source` le permite al daemon canjear `fsnotify` por una fuente de polling en filesystems donde inotify no es fiable (NFS, SMB, FUSE).

El modo daemon re-exec el mismo binario con env vars ocultos, así el mismo code path sirve ambos modos.

### `internal/supervisor` es dueño de los daemons

`internal/supervisor` es el proceso por usuario que es dueño de cada daemon de watch. Usa IPC por socket Unix para hablarle a la CLI, y el presupuesto de inotify para asegurar que ningún proyecto monopolicie las tablas de watches del kernel. Él mismo es supervisado por `internal/watchdog`.

## Concerns transversales

- **Sin CGo.** Mekami usa `modernc.org/sqlite`, un driver de SQLite Go puro. El binario es un único artefacto estático sin mismatch de ABI glibc/musl.
- **Testing stdlib puro.** Sin testify, sin ginkgo. `gofmt` + `go vet` para lint.
- **`mekami-api` stdlib puro.** El paquete `api/v1` no tiene dependencias internas, así que un frontend externo solo necesita depender de él para registrarse.

Mirá [Módulos](modules.md) para un tour por paquete, y [Soporte de plataforma](platform.md) para el split de service manager por OS.
