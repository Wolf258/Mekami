# Limitaciones

- **Un único frontend de lenguaje en el binario (Go).** La arquitectura soporta lenguajes adicionales (mirá [Escribir un frontend](extending/writing-a-frontend.md)), pero no se bundlea ningún frontend en el binario por defecto. Agregá un frontend publicando un nuevo módulo en `github.com/Wolf258/mekami-core-<lang>`, dependiendo de `github.com/Wolf258/mekami-api/api/v1`, y registrándolo vía `mekami core install <lang>`.
- **Sin texto de cuerpo en el índice.** Mekami indexa solo nombres de símbolos y aristas de referencia. Para buscar subcadenas dentro de cuerpos de funciones, comentarios, strings de log o TODOs, usá `mekami find-text` (regex server-side sobre el árbol de fuentes) o la herramienta de lectura de tu editor.
- **Resolución de tipos solo intra-procedural.** El resolver de tipos de variables locales entiende parámetros de función, declaraciones con `:=`, asignaciones planas, cláusulas `range` y llamadas a constructores mismo-paquete. No persigue a través de llamadas cross-package — eso requeriría `go/types` sobre el paquete completo, que está fuera de scope por ahora.
- **Builds de workspace vs. submódulo.** Compilar desde la raíz de un workspace indexa cada módulo `use`d; compilar desde un submódulo salte a los hermanos. Cambiar entre los dos sin `--clean` se rechaza para evitar dejar paths stale en la DB.
- **Sin daemon de background en `serve`.** `serve` corre una única sesión stdio por invocación; lee la base de datos pero nunca escribe en ella. La reindexación de larga duración se dispara explícitamente vía `build`, o en background vía el watcher que arranca `init --daemon=yes` o `start`. Múltiples instancias de `serve` sobre el mismo proyecto comparten el mismo daemon.

## Status

Etapa temprana. El esquema, el pipeline de ingest, el servidor MCP, la CLI y el suite de tests están en su lugar. Esperá cambios breaking mientras el toolset se expande y el resolver de tipos crece.
