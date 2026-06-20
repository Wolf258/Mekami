# Inicio rápido

Este walkthrough asume que tenés el binario `mekami` en tu `$PATH` (mirá [instalación](installation.md) si no).

## 1. Inicializar un workspace

Dentro de un proyecto Go, ejecutá:

```bash
mekami init
```

Esto crea un directorio `.mekami/` con un `config.json` por defecto y un placeholder `db.sqlite3`. La configuración se commitea por convención; la base de datos es por usuario y debería ir al `.gitignore`.

Si tu proyecto usa `go.work` y querés que cada módulo `use`d se indexe como un solo grafo, corré desde la raíz del workspace. Si querés que solo se indexe el módulo actual, corré desde adentro del directorio del módulo — Mekami autodetecta ambos casos.

## 2. Construir el índice

Una build one-shot:

```bash
mekami build
```

La primera build recorre cada archivo `*.go` bajo el workspace, lo parsea con `go/parser` y persiste los símbolos, definiciones, firmas y aristas de referencia en `.mekami/db.sqlite3`. Las corridas siguientes solo re-ingieren los archivos cuyo contenido o set de imports cambió.

Para repos grandes, la build es en paralelo con `runtime.NumCPU()` workers; las escrituras se serializan en una única transacción SQLite.

## 3. Hacer una pregunta

Los comandos de la CLI y las herramientas MCP de Mekami comparten vocabulario — cada consulta es un único comando.

```bash
# ¿Qué es Foo?
mekami find Foo

# ¿Quién llama a Bar?
mekami who-calls Bar

# ¿Cuál es el camino de llamada entre A y B?
mekami call-path A B

# Outlines
mekami file-outline ./cmd/...
mekami package-outline ./internal/foo
```

Todos los comandos leen de la misma `.mekami/db.sqlite3` que produjo la build. La CLI renderiza los resultados como texto legible; las mismas llamadas por MCP devuelven JSON.

## 4. Mantenerse en sync con el daemon

Para un flujo de trabajo dirigido por ediciones, corré el daemon de watch en lugar de `mekami build`:

```bash
mekami start
```

Esto lanza un daemon por proyecto que vigila los cambios, los debouncea, detecta cambios estructurales (p. ej. una edición a `go.mod`) y reindexa incrementalmente. Paralo con `mekami stop`.

Si querés que sobreviva las sesiones de shell, instalá el supervisor como servicio de usuario:

```bash
mekami service install --start
```

Mirá la página [Modo watch](../user-guide/watch-mode.md) para la historia completa de supervisor / watchdog / adopción de huérfanos.

## 5. Usarlo desde tu agente

Si conectaste Mekami a un cliente MCP con `mekami mcp install`, ahora podés pedirle al agente cosas como:

> ¿Quién llama a `connectToServer` y cuál es el camino de llamada desde `main` hasta ella?

El agente despachará las herramientas `who_calls` y `call_path` y sintetizará una respuesta desde el grafo. Mirá [Herramientas MCP](../user-guide/mcp-tools.md) para la superficie completa de herramientas.

## Siguiente paso

- Leé la [referencia CLI](../user-guide/cli.md) para ver todo lo que hay disponible.
- Leé [cómo funciona la indexación](../user-guide/how-it-works.md) para entender qué se persiste.
- Si querés agregar soporte para un nuevo lenguaje, mirá [extender Mekami](../extending/index.md).
