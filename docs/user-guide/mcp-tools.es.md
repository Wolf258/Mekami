# Herramientas MCP

El servidor MCP de Mekami expone 17 herramientas sobre stdio, derivadas del mismo slice `naming.Specs` que maneja la CLI. Los nombres de herramientas son snake_case; los comandos equivalentes de la CLI son kebab-case.

Todas las herramientas devuelven contenido de texto (JSON o texto formateado) sobre MCP. Las descripciones completas van embebidas en el servidor (el LLM las lee en cada llamada), así que la tabla de abajo es una referencia rápida.

| Herramienta | Propósito |
| --- | --- |
| `find_symbol` | Búsqueda por subcadena sobre nombres de símbolos. |
| `get_symbol` | La definición de un símbolo (texto formateado, legible). |
| `show_body` | El cuerpo del código de un símbolo (líneas numeradas). |
| `show_lines` | Un rango de líneas arbitrario de un archivo. |
| `who_calls` | Referencias entrantes (call, type-use, value, field, embed, import). |
| `what_calls` | Referencias salientes distintas. |
| `list_file` | Símbolos de primer nivel en un archivo. |
| `list_package` | Todos los símbolos de un paquete. |
| `show_modules` | Resumen de alto nivel de los módulos indexados y sus paquetes. |
| `list_modules` | Módulos indexados (JSON). |
| `list_package_symbols` | Símbolos de primer nivel declarados en un paquete dado. |
| `list_importers` | Paquetes que importan un paquete dado. |
| `list_files` | Árbol de archivos del proyecto desde el snapshot indexado. |
| `trace_calls` | BFS para encontrar un camino de llamada entre dos qualified names. |
| `show_changes` | Archivos agregados/modificados/eliminados desde el último `mekami build`. |
| `find_text` | Búsqueda regex server-side en los archivos fuente. |
| `index_status` | Snapshot del índice (`last_root`, `last_build_at`, conteos). |

## Filtros comunes

Varias herramientas aceptan filtros:

- `kind` (`func`, `type`, `method`, `var`, `const`) en `find_symbol` — filtra las clases de símbolos.
- `ref_kind` (`call`, `type-use`, `value`, `field`, `embed`, `import`) en `who_calls` — filtra las clases de aristas de referencia.
- `path_prefix` en la mayoría de las herramientas de listado — restringe a archivos cuyo path arranca con el prefijo dado.

## Sesión de ejemplo

> "¿Quién llama a `connectToServer` y cuál es el camino de llamada desde `main` hasta ella?"

El agente:

1. Llama `who_calls` con `qualified_name: "connectToServer"`.
2. Llama `trace_calls` con `from: "main"` y `to: "connectToServer"`.
3. Sintetiza una respuesta desde el grafo.

```text
call path: main -> server.Run -> runOnce -> connectToServer
  main                       cmd/example/main.go:42
  server.Run                 cmd/example/server.go:18
  runOnce                    internal/server/run.go:7
  connectToServer            internal/server/connect.go:31
```

## Smoke test

`mekami mcp test` lanza el servidor como subproceso y ejercita un muestreo de herramientas end-to-end contra el grafo indexado. Usalo después de cualquier cambio al servidor o después de un upgrade:

```bash
mekami build
mekami mcp test
```

El runner de smoke reporta el éxito/fallo de cada herramienta e imprime una línea de resumen. La salida con código distinto de cero significa que al menos una llamada a herramienta falló.
