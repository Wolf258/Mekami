# Cómo funciona la indexación

Cuando corrés `mekami build`, pasa lo siguiente, en orden.

## 1. Walk

El walker enumera cada archivo `*.go` bajo la raíz del código fuente, salteando `.git`, `.mekami`, `node_modules`, `vendor`, `_dev`, y `*_test.go` por defecto. Exclusiones adicionales vienen de `watch.ignore` en `.mekami/config.json`.

## 2. Fingerprint

Para cada archivo: leer los bytes, hashear con `sha256`, y comparar contra el hash almacenado. Los archivos sin cambios se saltean sin re-parsear. Esta es la base del build incremental.

## 3. Parse

Los archivos cambiados se parsean con `go/parser`. El collector recorre el AST y emite:

- **Símbolos** — `func`, `method`, `type`, `var`, `const`, más un ancla sintética `__imports__` para el bloque de imports. Cada símbolo lleva su `qualified_name` (p. ej. `graph.queries.SearchSymbols`), rango de líneas, firma, y estado de exportado.
- **Refs** — aristas `call`, `type-use`, `value` e `import`, cada una tageada con la línea de origen.

Un resolver de tipos intra-procedural ligero mapea variables locales a sus tipos declarados, así `m := recv.Field` puede resolver `Field` a `pkg.Type.Field` aún cuando el tipo del receiver está inferido. Los literales de función anónimos a nivel de archivo — la forma típica `&cobra.Command{ RunE: func(...) error { ... } }` — reciben un símbolo owner sintético (kind `funclit`, qualified name `pkg.__lit__<file>_<line>__`) para que cada llamada adentro del closure quede visible en `who-calls` y `trace`.

## 4. Write

Todos los resultados se escriben adentro de una única transacción SQLite (modo WAL, `synchronous=NORMAL`, `foreign_keys=ON`). Los archivos que desaparecieron desde el último build se eliminan en la misma pasada.

## Paralelismo

El parseo corre en `runtime.NumCPU()` workers. Cada worker es una llamada a `go/parser.ParseFile` contra su propio archivo; los resultados se streamean por un canal hacia un único writer que es dueño de la transacción SQLite. Esto te da speedup casi lineal en repos grandes mientras mantiene serializadas las escrituras a la base de datos.

## El modelo de datos

El store es intencionalmente angosto: cada fila está keyada en un `qualified_name` (para símbolos) o `(file, line, kind)` (para refs). El esquema es de `core/store/schema.go`; los tipos de fila y DTOs de `core/model/`. Para los tipos exactos, mirá la [referencia de la API](../api-reference/frontend-api.md).

## Lo que **no** se indexa

- **Texto del cuerpo del código fuente.** `mekami find-text` es una búsqueda regex aparte sobre el sistema de archivos, no parte del grafo. Mirá [Limitaciones](../limitations.md).
- **Resolución de tipos cross-package.** El resolver de variables locales entiende parámetros de función, declaraciones con `:=`, asignaciones planas, cláusulas `range` y llamadas a constructores mismo-paquete. No persigue a través de llamadas cross-package — eso requeriría `go/types` sobre el paquete completo, que está fuera de scope por ahora.
- **Archivos de test.** `*_test.go` se excluye por default. Si querés que se indexen, apuntá `--root` a un directorio que no los excluya o extendé el `IsStructuralFiles` / lista de exclusión del walker.

## Reindexado incremental

El watcher llama a un camino diferente: `BuildIncremental(paths)`. Carga el set de archivos actual desde la DB, computa el diff contra el set nuevo, y re-parsea solo los archivos cambiados o agregados. Los archivos eliminados se borran en la misma pasada. Un cambio en un archivo estructural (cualquiera de `go.mod` / `go.work` / `go.sum`) promueve el batch a una `Build` completa en su lugar.

El set de archivos estructurales es específico del frontend y se expone vía `Frontend.StructuralFiles()` (mirá [el contrato de frontend](../extending/frontend-contract.md)).
