# Contrato de frontend

El paquete `api/v1` (`github.com/Wolf258/mekami-api/api/v1`) es el contrato público que implementa cada indexer de lenguaje. El paquete solo depende de la stdlib de Go, así que los frontends externos solo necesitan depender de él para registrarse.

Para la referencia completa renderizada con godoc, mirá la [referencia del Frontend API](../api-reference/frontend-api.md).

## La interfaz `Frontend`

Un frontend es un paquete que se autoregistra y sabe cómo:

1. Identificar los archivos que reclama (`Extensions()`).
2. Resolver el layout del workspace para la raíz de la build (`ResolveLayout()`).
3. Enumerar cada módulo que la raíz de la build contiene (`ResolveModules()`).
4. Devolver el identifier canónico del módulo raíz (`RootModule()`).
5. Mapear un archivo a sus identifiers de módulo/paquete (`ResolveFile()`).
6. Parsear un único archivo a la forma genérica `ParseResult` (`ParseFile()`).
7. Listar los basenames cuya edición invalida todo el índice (`StructuralFiles()`).
8. Saltear tipos de archivo específicos del lenguaje del walk (`IsIndexable()`).
9. Reportar su identifier de lenguaje en minúsculas (`Name()`).

## Formas de datos

### `Workspace`

Describe un layout multi-módulo. Los indexers sin concepto de workspace devuelven `IsWorkspace: false` y un `WorkspaceMods` vacío.

| Campo | Descripción |
| --- | --- |
| `IsWorkspace` | True si el proyecto tiene un manifiesto de workspace. |
| `WorkFile` | Path absoluto del manifiesto (`go.work`, `Cargo.toml` workspace, …). |
| `WorkspaceDir` | Directorio que contiene el manifiesto. |
| `WorkspaceMods` | Paths absolutos de cada módulo `use`d / `member`. |
| `PrimaryModPath` | Path canónico del módulo primario. |
| `PrimaryModuleDir` | Path absoluto del módulo primario. |

### `FileMeta`

Información resuelta de paquete/módulo para un único archivo fuente.

| Campo | Descripción |
| --- | --- |
| `ModuleID` | Identifier de módulo agnóstico del lenguaje. |
| `PackageID` | Identifier de paquete agnóstico del lenguaje (dentro del módulo). |
| `DirRel` | Directorio del archivo relativo a la raíz de la build. |

### `ModuleInfo`

Un único módulo que el frontend descubrió en la raíz de la build. El resultado de `ResolveModules`.

| Campo | Descripción |
| --- | --- |
| `Dir` | Path absoluto del root del módulo. |
| `ModuleID` | Identifier canónico del módulo. |

### `ParseResult`

La salida CPU-only de parsear un único archivo. El pipeline de ingest lo escribe tal cual al store de SQLite; nada en el struct de resultado es específico del lenguaje.

| Campo | Descripción |
| --- | --- |
| `RelPath` | Path del archivo relativo a la raíz de la build. |
| `Lang` | Identifier estable en minúsculas. El pipeline lo usa para hacer short-circuit del re-ingest cuando cambia el lenguaje de un archivo. |
| `ModuleID` | El identifier de módulo del archivo. |
| `PackageID` | El identifier de paquete del archivo. |
| `DirRel` | Directorio del archivo relativo a la raíz de la build. |
| `Hash` | Hash SHA-256 del contenido del archivo. |
| `Mtime` | Tiempo de modificación del archivo. |
| `Size` | Tamaño del archivo en bytes. |
| `Symbols` | Símbolos declarados en este archivo. |
| `Refs` | Aristas de referencia originadas en este archivo. |

### `Symbol` y `SymbolKind`

Un `Symbol` es una única declaración. Los campos `file_id` y `package_id` los estampa el core después de que el indexer devuelve; el indexer debe dejarlos en cero.

| Campo | Descripción |
| --- | --- |
| `Kind` | Uno de `KindFunc`, `KindMethod`, `KindType`, `KindVar`, `KindConst`, `KindImports`, `KindFuncLit`. |
| `Name` | Nombre desnudo del símbolo. |
| `QualifiedName` | Forma `paquete.Símbolo`. |
| `StartLine` / `EndLine` | Rango de líneas 1-based (inclusivo). |
| `Exported` | True si el símbolo está exportado. |
| `Signature` | Firma renderizada para mostrar. |
| `ParentSymbol` | Puntero al id del símbolo padre (lo setea el core, no el indexer). |

`KindImports` es un ancla sintética para el bloque de imports del archivo. `KindFuncLit` es un owner sintético para nodos `*ast.FuncLit` de primer nivel (p. ej. `&cobra.Command{RunE: func(...) error {...}}`); asignarle un qualified name único mantiene cada llamada dentro del closure visible en `who_calls` y `trace_calls`.

### `Ref` y `RefKind`

Un `Ref` es una única arista de referencia. `FromSymbol` es el índice 0-based del símbolo originario dentro del mismo slice `ParseResult.Symbols`; el core lo traduce a un id real de la DB durante la fase de write.

| Campo | Descripción |
| --- | --- |
| `FromSymbol` | Índice 0-based en `ParseResult.Symbols`. |
| `ToQualified` | El qualified name que se está referenciando. |
| `Kind` | Uno de `RefCall`, `RefTypeUse`, `RefImport`, `RefValue`. |
| `Line` | Línea de origen 1-based. |

## El registry

`api.Global` es el registry por defecto. Los indexers llaman `api.Register(f)` desde una función `init()`; el `main` del binario blank-importa los paquetes de frontend que necesitan registrarse. Los nombres duplicados hacen panic al arranque, así un typo se caza temprano.

```go
func init() { api.Register(Frontend{}) }
```

## Garantías del contrato

Un frontend **DEBE**:

- Ser seguro para llamadas concurrentes a `ParseFile` / `ResolveFile` (el pipeline corre N workers).
- Devolver un slice `Symbols` y `Refs` no nulo (vacío está bien) — nunca `nil`. El writer los indexa directamente.
- Setear `ParseResult.Lang` a un identifier estable en minúsculas. El pipeline lo usa para hacer short-circuit del re-ingest cuando cambia el lenguaje de un archivo (p. ej. un `.go` se renombra a `.py`).
- Setear `Refs[i].FromSymbol` al índice 0-based del símbolo originario en `Symbols`. El writer lo traduce a un id real de la DB después de insertar los símbolos. El frontend Go usa índice 0-based; hacé lo mismo en el tuyo.

Un frontend **PUEDE**:

- Devolver un `Workspace` vacío desde `ResolveLayout` si el lenguaje no tiene concepto de workspace.
- Devolver `""` desde `RootModule` si el lenguaje no tiene un módulo raíz canónico.
- Devolver un `StructuralFiles` vacío si cada edición debería ser manejada por el camino incremental.

Un frontend **NO DEBERÍA**:

- Tocar la base de datos directamente. Persistí vía `api.ParseResult` y dejá que `ingest.WriteParseResult` maneje el write a la DB.
- Bloquearse en I/O fuera de su propio archivo. Los workers corren en paralelo y un frontend lento estancaría el pool.

## Compatibilidad de esquema

El store, las queries, las herramientas MCP y los DTOs son agnósticos del lenguaje. La tabla `packages` usa dos columnas de identifier:

- `module_id` — string que nombra al módulo del build (Go: `github.com/foo/bar`; Python: nombre de proyecto de `pyproject.toml`; Rust: nombre del crate).
- `package_id` — string que identifica al paquete dentro del módulo (Go: import path completo; Python: dotted module path; Rust: `crate::module`).

Los frontends que no tienen concepto de sub-paquete pueden usar `module_id` como `package_id` y el basename del directorio como columna `name`.
