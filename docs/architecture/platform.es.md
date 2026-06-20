# Soporte de plataforma

Mekami corre en Linux, macOS y Windows. El pipeline de indexación central es portable; las partes por OS se concentran en tres lugares.

## Integración con service manager

`mekami service install` registra el supervisor como servicio de sistema por usuario:

| OS | Backend | Archivo de unidad |
| --- | --- | --- |
| Linux | `systemd --user` | `~/.config/systemd/user/mekami-supervisor.service` |
| macOS | `launchd` (LaunchAgent) | `~/Library/LaunchAgents/dev.mekami.supervisor.plist` |
| Otro | no implementado | — |

Split del código:

- `cmd/mekami/service_linux.go` — `systemctl --user` enable/start.
- `cmd/mekami/service_darwin.go` — `launchctl bootstrap`/`kickstart`.
- `cmd/mekami/service_other.go` — retorna "no implementado".

En plataformas no soportadas el supervisor igual funciona: se lanza la primera vez que cualquier comando `mekami` lo necesita, y el watchdog lo mantiene vivo entre reboots. También podés correr el supervisor desde tu shell rc.

## Vigilancia del sistema de archivos

`internal/watch` usa `github.com/fsnotify/fsnotify` en cada plataforma. La abstracción `Source` le permite al daemon canjear `fsnotify` por una fuente de polling en filesystems donde inotify no es fiable:

- **Linux**: `inotify` vía `fsnotify`. Presupuesto de watches por usuario rastreado por el supervisor (default 8192 watches, degradado al 80% de uso al poller).
- **macOS**: `FSEvents` vía `fsnotify`. Sin presupuesto por usuario; el fallback al poller es solo para mounts NFS / SMB / FUSE.
- **Windows**: `ReadDirectoryChangesW` vía `fsnotify`. Mismo fallback para NFS / SMB / FUSE.
- **NFS / SMB / FUSE** (cualquier OS): auto-detectado y cambiado al poller (`fallback: "auto"`).

## SQLite

`modernc.org/sqlite` es Go puro, así que el mismo binario corre en cada plataforma soportada sin CGo. El driver se carga en tiempo de link y se bundlea en el binario.

## Matriz de CI probada

`.github/workflows/mekami.yml` corre en cada push a `main` y en cada pull request, en:

- `ubuntu-latest`
- `macos-latest`
- `windows-latest`

con Go 1.26.

## Quirks conocidos por plataforma

- **Límite de path de socket en macOS.** Los Unix sockets en macOS están limitados a 104 bytes para `sun_path`. El paquete `internal/testutil` exporta `ShortSockDir` para que los tests puedan usar un path más corto.
- **Service manager en Windows.** Windows está soportado para la CLI core y el modo `serve`, pero `service install` no está implementado. Corré el supervisor desde una scheduled task o manualmente.
