# Watch mode

`mekami start` keeps the index in sync with the source tree while you edit. The watcher is a long-lived daemon owned by a per-user **supervisor** process. There is at most one supervisor per user, and it manages every Mekami daemon across every project you have initialised.

## Lifecycle at a glance

```bash
mekami init --daemon=yes          # create config, build, start daemon
# or, manually:
mekami start                      # ask the supervisor to spawn a daemon
mekami status                     # one-line summary
mekami logs                       # tail the daemon log
mekami stop                       # ask the supervisor to stop the daemon
mekami restart                    # stop + start
mekami reload                     # re-read .mekami/config.json
```

`mekami service install` registers the supervisor as a system service (per-user, single instance) so it starts automatically when you log in and rehydrates every daemon from `daemons.json`:

- **Linux**: writes a single `systemd --user` unit (`mekami-supervisor.service`).
- **macOS**: writes a single `~/Library/LaunchAgents` plist (`dev.mekami.supervisor`).
- **Other platforms**: not implemented (you can still run the supervisor manually from your shell rc). The watchdog still works in this mode: it spawns alongside the supervisor and gives you auto-restart of the supervisor for free, even without a service manager.

## The supervisor

The supervisor is the per-user process that owns all watcher daemons. It:

- starts/stops daemons on demand (`init --daemon=yes`, `start`, `stop`, `restart`),
- monitors each daemon and restarts it on crash (with backoff),
- re-reads `daemons.json` on startup and rehydrates every daemon that was active before the supervisor stopped,
- **adopts orphaned daemons** that survived a supervisor crash (PID + socket + ping) instead of double-forking,
- tracks the global inotify watch budget and degrades the noisiest daemons to the poller when the budget gets tight,
- is itself supervised by a tiny **watchdog** process that re-spawns the supervisor when it is wedged (PID alive but unresponsive).

State lives in `$XDG_CONFIG_HOME/mekami/supervisor/`:

- `daemons.json` — the registered daemons and their last known state.
- `supervisor.sock` — the Unix socket the CLI talks to.
- `supervisor.pid` — the supervisor's PID (single-instance).
- `supervisor.log` — the supervisor's own log.

You rarely invoke the supervisor directly; the daemon commands do it for you. The supervisor is what `init --daemon=yes` and `start` start on first use.

## Orphan adoption

When the supervisor starts up (or when a user runs `mekami start` manually), it checks every project in `daemons.json` whose last known state was `running`, `starting`, `reloading`, or `crashed`. For each, it asks:

1. Is `.mekami/watcher.pid` present and parseable?
2. Is the recorded PID alive (`kill -0`)?
3. Does `.mekami/watcher.sock` exist?
4. Does that socket answer a `ping`?

If all four answers are yes, the existing daemon is **adopted**: the supervisor records its PID in its in-memory table and skips the re-spawn. This is what makes `kill -9 mekami-supervisor` safe — the watcher keeps running, the next supervisor invocation finds it, and you do not end up with two daemons fighting over the same project socket.

If the PID file is stale (the recorded process is gone) but the socket is still there, `Start` cleans up `.mekami/` (pids/socket/heartbeat) before forking a fresh daemon. The cleanup is best-effort: a leftover file that cannot be removed is reported as a normal spawn error.

If the heartbeat file is present but stale at adoption time (more than 30s since the last write), the supervisor logs a warning to `supervisor.log` but still adopts the daemon. A PID that responds to `kill -0` and answers a ping is, by definition, alive; the heartbeat may just be lagging.

## The supervisor watchdog

A daemon that lives only as long as its supervisor is fragile: if the supervisor ever wedges (alive in the process table, but not responding to its IPC socket), nothing on the system will restart it. `systemd --user` and `LaunchAgents` only restart a process that has exited; they cannot tell that a process is stuck.

To close this gap, the supervisor is launched together with a tiny sibling: the **watchdog**. The watchdog polls the supervisor's PID and Unix socket every 5 seconds. After 6 consecutive failed health checks (30 seconds of unresponsiveness), the watchdog:

1. Sends `SIGKILL` to the supervisor's PID.
2. Removes the stale `supervisor.sock` so the new supervisor can bind it.
3. Re-spawns the supervisor (`supervise _run`), which in turn re-spawns its own watchdog.

The watchdog is best-effort:

- If the supervisor exits cleanly, the watchdog notices the missing PID file and exits; the service manager (`systemd --user` / `LaunchAgent`) restarts the whole pair. The watchdog is not a replacement for the service install; it is a complement that catches the "wedged but alive" case the service manager cannot.
- If you do not run `mekami service install`, the watchdog still works: it is launched automatically the first time any `mekami` command needs the supervisor. The watchdog is what keeps the supervisor alive across reboots on platforms without a service manager.

You never invoke the watchdog directly. It is the hidden `supervise _watchdog` subcommand and runs in its own session (`setsid`) so it survives the parent shell exiting. On startup the watchdog writes its own PID to `$XDG_CONFIG_HOME/mekami/supervisor/watchdog.pid` and removes the file on exit, so `service uninstall` can find and signal it without scanning the process table.

The watchdog also watches for a **stop sentinel** at `$XDG_CONFIG_HOME/mekami/supervisor/stop`. When the file is present, the watchdog exits on its next tick (immediately if the sentinel is already there on startup) regardless of supervisor state. The sentinel is what `service uninstall` uses to make the watchdog exit deterministically rather than waiting for the next health-check tick to discover the supervisor is gone. The supervisor clears the sentinel on the next startup so a leftover file from a previous uninstall does not cascade into the new run.

## Daemon health and orphan recovery

Each watcher daemon writes a heartbeat to `.mekami/heartbeat` every 5 seconds. The heartbeat is a single line containing the unix-nano timestamp of the write. The supervisor uses it as a secondary liveness signal: a daemon that answers `kill -0` and pings but has not refreshed its heartbeat in 30 seconds is logged as "stale heartbeat" on adoption, so a future maintainer can see whether a previously-frozen process was picked up.

The daemon also carries a copy of the supervisor's PID (the `_MEKAMI_DAEMON_SUPERVISOR_PID` env var). It pings that PID every 5 seconds; if the supervisor becomes unreachable, the daemon logs `"warning: supervisor pid=N unreachable, running standalone"` once a minute. By default the daemon keeps running — losing the supervisor is not a reason to lose the index.

If you want the daemon to give up after being orphaned for a while (for example, in CI containers that come and go), set `watch.self_terminate_on_orphan` in `.mekami/config.json`:

```json
{
  "watch": {
    "self_terminate_on_orphan": "10m"
  }
}
```

The value is a `time.ParseDuration` string (`30s`, `5m`, `1h`, ...). The empty string (the default) means "never self-terminate", which is the right default for developers who want the watcher to keep the index fresh even when no supervisor is around.

## The inotify budget

The inotify budget is enforced on Linux. Each fsnotify watcher registers one watch per directory; with thousands of directories across many projects, the per-user limit (`/proc/sys/fs/inotify/max_user_watches`, typically 8192 by default) gets tight. The supervisor measures consumption; once it crosses 80% it flips the noisiest daemons to the poller (`fallback: "poll"`) automatically. If you want to raise the limit:

```bash
# Raise the per-user watch budget to 524288.
sudo sysctl fs.inotify.max_user_watches=524288
```

## Uninstalling the service

`mekami service uninstall` is the symmetric counterpart to `service install`. On Linux and macOS it:

1. Sends a `quit-all` IPC request to the running supervisor. The supervisor stops every registered daemon (graceful IPC stop → `SIGTERM` → `SIGKILL` on timeout), writes the stop sentinel, and signals the watchdog's PID file so the watchdog exits immediately on its next tick.
2. Sends `SIGTERM` via the service manager (`systemctl --user disable --now` on Linux, `launchctl unload -w` on macOS) as a safety net for the case where the supervisor was not running or its IPC socket was unreachable.
3. Removes the runtime state files from `$XDG_CONFIG_HOME/mekami/supervisor/`: `supervisor.pid`, `supervisor.sock`, `supervisor.log`, `watchdog.pid`, and the stop sentinel. A missing file is not an error; a permission error is logged but does not abort the uninstall.
4. Removes the unit file (`mekami-supervisor.service` on Linux, `dev.mekami.supervisor.plist` on macOS) and tells the service manager to reload.

The per-project `.mekami/` directories and the `daemons.json` registry are **preserved**. A subsequent `mekami service install` will rehydrate the same set of daemons from the registry, so the user's intent ("watch these projects") survives the uninstall. The result is what we call a **hard uninstall**: the supervisor, watchdog, and all daemon children are gone, but the registry and per-project state are intact. A future install brings everything back as it was.

If you also want the registry and per-project state removed, the user can do it manually (`rm -rf $XDG_CONFIG_HOME/mekami` and the `.mekami/` directories inside each project). Adding a `--purge` flag to `service uninstall` is a deliberate non-feature: deleting user data without an explicit, separate opt-in is too easy to do by accident.
