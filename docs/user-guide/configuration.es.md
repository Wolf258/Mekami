# Configuración

Mekami lee su configuración de `.mekami/config.json`. El archivo es opcional; ausente → defaults razonables.

## Esquema

```json
{
  "version": 1,
  "watch": {
    "enabled": true,
    "debounce_ms": 250,
    "ignore": ["*.tmp", "*.swp", ".DS_Store"],
    "on_start": "build",
    "log": "info",
    "fallback": "auto",
    "poll_interval_s": 30,
    "log_level": "resumen",
    "self_terminate_on_orphan": ""
  },
  "build": {
    "jobs": 0
  },
  "indexers": {
    "go": "v0.1.0"
  }
}
```

## `indexers`

Un mapa del nombre del lenguaje a la versión que resolvió `core install`.

```json
"indexers": {
  "go": "v0.1.0",
  "rust": ""
}
```

Un valor vacío (`"rust": ""`) significa que el lenguaje fue agregado por `mekami init` pero todavía no se corrió `core install` para él — la build igual lo rastrea, y una `mekami build --lang rust` posterior (o `core install rust`) completa la versión.

`core install <lang>[@<version>]` resuelve la versión vía el proxy de módulos de Go (`go list -m -versions`), escribe la entrada en `indexers` y regenera `mekami-cli/internal/core/frontend/all_gen/all_gen.go` con un blank import nuevo.

`mekami core list` y `core status` muestran el conjunto de indexers pedido por la config versus lo que el binario en ejecución tiene registrado. Los frontends que están listados pero cuyo blank import falta se reportan como `missing`.

## Opciones de `watch`

| Clave | Default | Descripción |
| --- | --- | --- |
| `enabled` | `true` | Desactivá para que el daemon salga inmediatamente al arrancar (útil para actualizaciones de índice one-shot). |
| `debounce_ms` | `250` | Ventana de silencio que el coalescer espera tras el último evento de FS antes de disparar un rebuild. `0` desactiva el debounce. |
| `ignore` | `[]` | Globs de basename que se descartan por encima de las exclusiones built-in del walker de build (`.git`, `.mekami`, `vendor`, `node_modules`, `_dev`). |
| `on_start` | `"build"` | Qué hace el watcher una vez antes de entrar al event loop: `build` (Build completo), `incremental`, o `skip`. |
| `log` | `"info"` | Uno de `info` (una línea por batch), `debug` (por evento), o `quiet` (solo errores). Aplica a la CLI en foreground. |
| `fallback` | `"auto"` | Fuente de eventos: `auto` (fsnotify + fallback a poller), `fsnotify` (forzar inotify), o `poll` (forzar poller). |
| `poll_interval_s` | `30` | Cadencia del polling cuando `fallback` es `poll` o el FS se detecta como poco fiable. |
| `log_level` | `"resumen"` | Log persistido del daemon: `resumen` (una línea por batch) o `verbose` (por evento). Rotado a 1 MiB con tres backups. |
| `self_terminate_on_orphan` | `""` | String `time.ParseDuration` (`"30s"`, `"10m"`, `"1h"`). Vacío = nunca se auto-termina. Mirá [Modo watch](watch-mode.md#salud-del-daemon-y-recuperacion-de-huerfanos). |

## Opciones de `build`

| Clave | Default | Descripción |
| --- | --- | --- |
| `jobs` | `0` | Workers de parseo. `0` significa `runtime.NumCPU()`. |

## Cambios estructurales

Cualquier edición a `go.mod`, `go.work` o `go.sum` promueve automáticamente el batch actual a un rebuild completo — el camino incremental del watcher es para ediciones ordinarias de código Go, donde re-recorrer el árbol es trabajo desperdiciado. Si el watcher no encuentra `last_root` en la DB (p. ej. corriste `start` sin haber hecho nunca un build), cae a un build one-shot completo.

El conjunto de archivos "estructurales" se configura por frontend vía `Frontend.StructuralFiles()` (mirá [el contrato de frontend](../extending/frontend-contract.md)).

## Apagado limpio

El daemon de watch se apaga limpiamente con `SIGINT` / `SIGTERM` (enviado vía `mekami stop`) y escribe una línea de resumen final con contadores de batch / archivo / error a su log.
