# Módulos

El repo está dividido en tres módulos Go. Un workspace (`go.work`) los ata juntos para desarrollo local.

## `github.com/Wolf258/mekami-api`

Módulo tiny de stdlib puro que define la interfaz `api.Frontend` que implementa cada indexer de lenguaje. Cero dependencias internas — los frontends externos solo necesitan depender de este.

Tipos y funciones públicas:

- `Workspace`, `FileMeta`, `ModuleInfo`
- `ParseResult`, `Symbol`, `SymbolKind`
- `Ref`, `RefKind`
- `ModuleEntry`
- Interfaz `Frontend` (9 métodos)
- `Registry` y el registro global `api.Global`
- `Register`, `Get`, `Names`, `All`, `IsStructural`, `DefaultStructuralFiles`

Para la referencia completa, mirá la [referencia del Frontend API](../api-reference/frontend-api.md).

## `github.com/Wolf258/mekami-core-go`

Módulo standalone de frontend para Go. Implementa `api.Frontend`. Vive en su propio repo para que otros lenguajes puedan seguir la misma forma (`mekami-core-rust`, `mekami-core-c`, …).

Archivos:

- `parser.go` — entry point y `ParseFile`.
- `collector.go` — walker de alto nivel.
- `visitor.go` — `ast.Visitor` que emite filas `Symbol` y `Ref`.
- `walkexpr.go` — resolver de tipos intra-procedural.
- `imports.go` — manejo del bloque de imports y ancla sintética `__imports__`.
- `resolve.go` — resolución de paquetes e imports.
- `astutil.go` — helpers chiquitos de AST (render de firma, armado de qualified-name, sint de funclit).

## `github.com/Wolf258/mekami-cli`

La CLI / MCP / supervisor / daemon. Módulo único. Blank-importa `internal/core/frontend/all_gen` para registrar los frontends compilados dentro.

```text
mekami-cli/
├── main.go                              # blank-imports all_gen
├── go.mod
├── cmd/mekami/                          # entrypoint de cobra
│   ├── root.go                          # Specs -> loop de cobra
│   ├── runner.go                        # dispatch + --json + exit codes
│   ├── commands.go                      # lifecycle / daemon / mcp runners
│   ├── coreinstall.go                   # core install / list / uninstall / status
│   ├── mcptest.go                       # smoke runner de mekami mcp test
│   ├── util.go                          # printJSON, helpers de supervisor
│   ├── service_linux.go                 # systemd --user
│   ├── service_darwin.go                # LaunchAgent
│   ├── service_other.go                 # stub
│   ├── service_status.go                # runner de service status
│   └── dbpath.go                        # plumbing del flag --db
├── internal/
│   ├── config/                          # esquema de .mekami/config.json + Load
│   ├── coreinstall/                     # core install / list / uninstall
│   ├── naming/                          # única fuente de verdad (Spec, Specs)
│   ├── handlers/                        # implementaciones de lectura compartidas
│   ├── mcp/                             # servidor MCP, tool registry desde Specs
│   ├── format/                          # formatters de texto legible
│   ├── install/                         # registro de cliente MCP (opencode)
│   ├── watch/                           # daemon de watch
│   ├── supervisor/                      # supervisor de daemons por usuario
│   ├── watchdog/                        # watchdog del supervisor
│   ├── core/                            # pipeline de indexación mergeado
│   │   ├── ingest/                      # build / incremental / write
│   │   ├── store/                       # store de SQLite
│   │   ├── queries/, path/, diff/, grep/  # helpers del lado lectura
│   │   ├── walk/                        # walker de FS + fingerprint
│   │   ├── modlayout/                   # resolución de go.mod / go.work
│   │   ├── model/                       # filas de DB + DTOs
│   │   ├── frontend/
│   │   │   ├── all_gen/                 # blank imports generados
│   │   │   └── README.md                # cómo escribir un nuevo indexer
│   │   └── integration_test/            # tests e2e, build tag "integration"
│   └── testutil/                        # helpers de tests cross-package
└── tests/                               # tests black-box
```

## Grafo de dependencias en capas

```text
                  ┌─────────────────────┐
                  │      cmd/mekami     │
                  └──────────┬──────────┘
                             │
       ┌─────────────────────┼─────────────────────┐
       ▼                     ▼                     ▼
┌─────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  internal/  │    │   internal/mcp   │    │   internal/     │
│  watch      │    │                  │    │   supervisor    │
└──────┬──────┘    └────────┬────────┘    └────────┬────────┘
       │                    │                      │
       └────────┬───────────┴──────────────────────┘
                ▼
        ┌──────────────────┐
        │  internal/       │
        │  handlers        │
        └────────┬─────────┘
                 ▼
        ┌──────────────────┐         ┌────────────────────┐
        │  internal/core/  │ ◀──────▶│ mekami-api/api/v1  │
        │  ingest / store  │         └────────────────────┘
        └──────────────────┘
                 ▲
                 │ (blank import vía all_gen)
                 │
        ┌────────┴─────────┐
        │ mekami-core-go   │
        │ (y futuros lang) │
        └──────────────────┘
```

`cmd/mekami` es el único paquete que depende de `cobra`. El resto es una librería pura.
