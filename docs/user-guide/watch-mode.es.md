# Modo watch

`mekami start` mantiene el índice sincronizado con el árbol de fuentes mientras editás. El watcher es un daemon de larga duración propiedad de un proceso **supervisor** por usuario. Hay a lo sumo un supervisor por usuario, y maneja todos los daemons de Mekami de todos los proyectos que inicializaste.

## Lifecycle de un vistazo

```bash
mekami init --daemon=yes          # crea config, build, inicia daemon
# o, manualmente:
mekami start                      # le pide al supervisor que lance un daemon
mekami status                     # resumen de una línea
mekami logs                       # tail del log del daemon
mekami stop                       # le pide al supervisor que pare el daemon
mekami restart                    # stop + start
mekami reload                     # re-lee .mekami/config.json
```

`mekami service install` registra el supervisor como servicio de sistema (por usuario, instancia única) para que se inicie automáticamente al loguearte y rehidrate todos los daemons desde `daemons.json`:

- **Linux**: escribe una única unidad `systemd --user` (`mekami-supervisor.service`).
- **macOS**: escribe un único plist `~/Library/LaunchAgents` (`dev.mekami.supervisor`).
- **Otras plataformas**: no implementado (podés correr el supervisor manualmente desde tu shell rc). El watchdog igual funciona en este modo: arranca junto al supervisor y te da auto-restart del supervisor gratis, incluso sin un service manager.

## El supervisor

El supervisor es el proceso por usuario que es dueño de todos los daemons de watch. Él:

- inicia/para daemons bajo demanda (`init --daemon=yes`, `start`, `stop`, `restart`),
- monitorea cada daemon y lo reinicia si crashea (con backoff),
- re-lee `daemons.json` al arrancar y rehidrata cada daemon que estaba activo antes de que el supervisor parara,
- **adopta daemons huérfanos** que sobrevivieron a un crash del supervisor (PID + socket + ping) en lugar de hacer doble fork,
- rastrea el presupuesto global de inotify y degrada los daemons más ruidosos al poller cuando el presupuesto se ajusta,
- es supervisado a su vez por un pequeño proceso **watchdog** que re-lanza el supervisor cuando queda colgado (PID vivo pero no responde).

El estado vive en `$XDG_CONFIG_HOME/mekami/supervisor/`:

- `daemons.json` — los daemons registrados y su último estado conocido.
- `supervisor.sock` — el socket Unix al que habla la CLI.
- `supervisor.pid` — el PID del supervisor (instancia única).
- `supervisor.log` — el log propio del supervisor.

Rara vez invocás al supervisor directamente; los comandos de daemon lo hacen por vos. El supervisor es lo que `init --daemon=yes` y `start` arrancan en el primer uso.

## Adopción de huérfanos

Cuando el supervisor arranca (o cuando un usuario corre `mekami start` manualmente), revisa cada proyecto en `daemons.json` cuyo último estado conocido fue `running`, `starting`, `reloading` o `crashed`. Para cada uno, pregunta:

1. ¿Está `.mekami/watcher.pid` presente y parseable?
2. ¿El PID registrado está vivo (`kill -0`)?
3. ¿Existe `.mekami/watcher.sock`?
4. ¿Ese socket responde a un `ping`?

Si las cuatro respuestas son sí, el daemon existente es **adoptado**: el supervisor registra su PID en su tabla en memoria y se saltea el re-launch. Esto es lo que hace seguro `kill -9 mekami-supervisor` — el watcher sigue corriendo, la próxima invocación del supervisor lo encuentra, y no terminás con dos daemons peleándose por el mismo socket del proyecto.

Si el archivo de PID está stale (el proceso registrado ya no está) pero el socket sigue ahí, `Start` limpia `.mekami/` (pids/socket/heartbeat) antes de forkear un daemon nuevo. La limpieza es best-effort: un archivo leftover que no se puede eliminar se reporta como un error normal de spawn.

Si el archivo de heartbeat está presente pero stale al momento de adoptar (más de 30s desde la última escritura), el supervisor loguea un warning a `supervisor.log` pero igual adopta el daemon. Un PID que responde a `kill -0` y contesta un ping está, por definición, vivo; el heartbeat puede estar simplemente atrasado.

## El watchdog del supervisor

Un daemon que vive solo tanto como su supervisor es frágil: si el supervisor alguna vez queda colgado (vivo en la tabla de procesos, pero sin responder a su socket IPC), nada en el sistema lo va a reiniciar. `systemd --user` y `LaunchAgents` solo reinician un proceso que haya salido; no pueden darse cuenta de que un proceso está colgado.

Para cerrar este gap, el supervisor se lanza junto con un hermano pequeño: el **watchdog**. El watchdog polea el PID del supervisor y el socket Unix cada 5 segundos. Tras 6 health checks fallidos consecutivos (30 segundos de no respuesta), el watchdog:

1. Envía `SIGKILL` al PID del supervisor.
2. Elimina el `supervisor.sock` stale para que el nuevo supervisor pueda bindearlo.
3. Re-lanza el supervisor (`supervise _run`), que a su vez re-lanza su propio watchdog.

El watchdog es best-effort:

- Si el supervisor sale limpio, el watchdog nota el archivo de PID faltante y sale; el service manager (`systemd --user` / `LaunchAgent`) reinicia el par. El watchdog no es un reemplazo del service install; es un complemento que cubre el caso "colgado pero vivo" que el service manager no puede.
- Si no corrés `mekami service install`, el watchdog igual funciona: se lanza automáticamente la primera vez que cualquier comando `mekami` necesita al supervisor. El watchdog es lo que mantiene vivo al supervisor entre reboots en plataformas sin service manager.

Nunca invocás al watchdog directamente. Es el subcomando oculto `supervise _watchdog` y corre en su propia sesión (`setsid`) para que sobreviva a que la shell padre salga. Al arrancar el watchdog escribe su propio PID en `$XDG_CONFIG_HOME/mekami/supervisor/watchdog.pid` y elimina el archivo al salir, así `service uninstall` puede encontrarlo y señalizarlo sin escanear la tabla de procesos.

El watchdog también mira un **sentinela de stop** en `$XDG_CONFIG_HOME/mekami/supervisor/stop`. Cuando el archivo está presente, el watchdog sale en su próximo tick (inmediatamente si la sentinela ya estaba al arrancar) sin importar el estado del supervisor. La sentinela es lo que usa `service uninstall` para que el watchdog salga determinísticamente en vez de esperar al próximo tick de health check a descubrir que el supervisor se fue. El supervisor limpia la sentinela en el próximo arranque así un archivo leftover de un uninstall anterior no se propaga al nuevo run.

## Salud del daemon y recuperación de huérfanos

Cada daemon de watch escribe un heartbeat a `.mekami/heartbeat` cada 5 segundos. El heartbeat es una única línea con el timestamp unix-nano de la escritura. El supervisor lo usa como señal de vida secundaria: un daemon que contesta `kill -0` y pings pero no refrescó su heartbeat en 30 segundos se loguea como "stale heartbeat" al adoptar, para que un futuro mantenedor pueda ver si un proceso previamente congelado fue recogido.

El daemon también lleva una copia del PID del supervisor (la env var `_MEKAMI_DAEMON_SUPERVISOR_PID`). Pingea ese PID cada 5 segundos; si el supervisor se vuelve inalcanzable, el daemon loguea `"warning: supervisor pid=N unreachable, running standalone"` una vez por minuto. Por defecto el daemon sigue corriendo — perder el supervisor no es razón para perder el índice.

Si querés que el daemon se dé por vencido tras estar huérfano un rato (por ejemplo, en containers de CI que van y vienen), seteá `watch.self_terminate_on_orphan` en `.mekami/config.json`:

```json
{
  "watch": {
    "self_terminate_on_orphan": "10m"
  }
}
```

El valor es un string `time.ParseDuration` (`30s`, `5m`, `1h`, ...). El string vacío (el default) significa "nunca se auto-termina", que es el default correcto para developers que quieren que el watcher mantenga fresco el índice aún cuando no haya supervisor cerca.

## El presupuesto de inotify

El presupuesto de inotify se enforce en Linux. Cada watcher de fsnotify registra un watch por directorio; con miles de directorios repartidos en muchos proyectos, el límite por usuario (`/proc/sys/fs/inotify/max_user_watches`, típicamente 8192 por defecto) se ajusta. El supervisor mide el consumo; una vez que cruza el 80% automáticamente flipea los daemons más ruidosos al poller (`fallback: "poll"`). Si querés subir el límite:

```bash
# Sube el presupuesto por usuario de watches a 524288.
sudo sysctl fs.inotify.max_user_watches=524288
```

## Desinstalar el servicio

`mekami service uninstall` es la contraparte simétrica de `service install`. En Linux y macOS:

1. Envía una solicitud IPC `quit-all` al supervisor en ejecución. El supervisor para cada daemon registrado (IPC stop gracioso → `SIGTERM` → `SIGKILL` por timeout), escribe la sentinela de stop, y le señaliza al archivo PID del watchdog para que el watchdog salga inmediatamente en su próximo tick.
2. Envía `SIGTERM` vía el service manager (`systemctl --user disable --now` en Linux, `launchctl unload -w` en macOS) como red de seguridad para el caso de que el supervisor no estuviera corriendo o su socket IPC no respondiera.
3. Elimina los archivos de estado de runtime de `$XDG_CONFIG_HOME/mekami/supervisor/`: `supervisor.pid`, `supervisor.sock`, `supervisor.log`, `watchdog.pid`, y la sentinela de stop. Un archivo faltante no es error; un error de permiso se loguea pero no aborta el uninstall.
4. Elimina el archivo de unidad (`mekami-supervisor.service` en Linux, `dev.mekami.supervisor.plist` en macOS) y le dice al service manager que recargue.

Los directorios `.mekami/` por proyecto y el registro `daemons.json` **se preservan**. Un `mekami service install` posterior rehidratará el mismo conjunto de daemons desde el registro, así que la intención del usuario ("mirar estos proyectos") sobrevive al uninstall. El resultado es lo que llamamos un **hard uninstall**: el supervisor, watchdog y todos los hijos daemon se fueron, pero el registro y el estado por proyecto están intactos. Un install futuro trae todo de vuelta como estaba.

Si también querés que se elimine el registro y el estado por proyecto, el usuario puede hacerlo manualmente (`rm -rf $XDG_CONFIG_HOME/mekami` y los directorios `.mekami/` dentro de cada proyecto). Agregar un flag `--purge` a `service uninstall` es un no-feature deliberado: borrar datos del usuario sin un opt-in explícito y separado es muy fácil de hacer por accidente.
