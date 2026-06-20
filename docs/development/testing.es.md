# Testing

Cómo está organizado el suite de tests, cómo correrlo, y las convenciones que los contribuidores deben seguir al agregar tests nuevos.

## De un vistazo

- 50 archivos `*_test.go`, 314 funciones de test, sin dependencias externas de test framework (sin `testify`, sin `ginkgo`).
- Dos niveles: **unit** (default `go test`) e **integration** (build tag `integration`).
- Módulos Go en este repo:
    - `mekami-cli/` (en este workspace, primario; contiene `internal/core/` que solía ser el módulo standalone `mekami-core`)
    - `mekami-api/` (externo, traído del proxy de módulos)
    - `mekami-core-go/` (externo, traído del proxy de módulos)

## Corriendo el suite

### Tests unitarios (default)

Desde la raíz del repo, con el `go.work` commiteado:

```bash
go env GOWORK
go test -short ./mekami-cli/...
```

`go test -short ./...` desde la raíz del repo es rechazado por Go porque el directorio raíz no es un módulo en sí — el workspace solo lista `./mekami-cli` como módulo. Pasá el patrón explícitamente.

Esto es lo que corre CI y lo que corre `check()` de AUR. Es rápido (segundos), hermético, y ejercita cada paquete excepto los gateados detrás del build tag `integration`.

Para correr un único paquete:

```bash
go test -count=1 ./mekami-cli/cmd/mekami/
go test -count=1 -run '^TestResolveLang$' ./mekami-cli/cmd/mekami/
```

Para correr con el detector de races:

```bash
go test -race ./mekami-cli/...
```

### Tests de integración

Los tests de integración viven detrás del build tag `integration`. El `go test` default no los compila.

Para la mayoría de los tests de integración no necesitás un clon local de los cores externos — son Go puro y corren contra `mekami-core-go` vía el proxy de módulos:

```bash
go test -tags integration ./mekami-cli/internal/core/integration_test/...
go test -tags integration ./mekami-cli/internal/watch/...
```

El round-trip con el service manager en `service_integration_test.go` también vive en el mismo módulo y usa el mismo build tag `integration`.

## Build tags

| Tag | Archivos | Propósito |
|---|---|---|
| `integration` | 20 en `mekami-cli/internal/core/integration_test/`, 1 en `mekami-cli/internal/watch/integration_test.go`, 1 en `mekami-cli/cmd/mekami/service_integration_test.go` | Tests end-to-end que necesitan un parser real de `mekami-core-go`, el sistema de archivos, o un user bus vivo. |
| `integration && linux` | `mekami-cli/cmd/mekami/service_integration_test.go` | Round-trip del service manager; depende de un user bus de systemd vivo, así que es solo Linux. |
| `!integration` | `mekami-cli/internal/core/ingest_test/{setup,stub_frontend}_test.go` | Lo opuesto a `integration`. Cablea el frontend Go stub para que los tests unitarios corran sin el paquete real `mekami-core-go`. |

La forma del build tag es la moderna `//go:build` (Go 1.17+). No mantenemos la forma legacy `// +build`; el proyecto targetea Go 1.26 y no hay razón de compatibilidad.

## Matriz de tests

| Módulo | Unit | Integration | Notas |
|---|---|---|---|
| `mekami-cli/internal/core/store` | sí | — | Round-trips de upsert / upsert-parent. |
| `mekami-cli/internal/core/queries` | sí | — | Helper de query de stats. |
| `mekami-cli/internal/core/path` | sí | — | Tests de tabla de error-wrap. |
| `mekami-cli/internal/core/grep` | sí | — | Matcher de grep. |
| `mekami-cli/internal/core/ingest_test` | sí (`!integration`) | — | Frontend stub, hermético. |
| `mekami-cli/internal/core/integration_test` | — | sí (20) | `mekami-core-go` real, grafo completo, prune, refs, mcp polish, etc. |
| `mekami-cli/internal/core/scripts/dev-allgen` | sí | — | Regenerador de `all_gen.go`. |
| `mekami-cli/cmd/mekami` | sí | sí (`integration && linux`) | resolveLang, resolveInitLangs, mergeIndexers, runInit, runBuild, comandos de service. |
| `mekami-cli/internal/config` | sí | — | Default, Load, Validate, OnStartAction, ShouldLog, Indexers. |
| `mekami-cli/internal/coreinstall` | sí | — | SplitLangRef, IsValidLang, NormalizeVersion, HighestVersion, List, Gen. |
| `mekami-cli/internal/handlers` | sí | — | Handlers de lectura (show_body, show_changes, list_package, find_symbol, who_calls, trace_calls, find_text). |
| `mekami-cli/internal/supervisor` | sí | — | state machine del supervisor, watchdog, spawn, registry, ipc, presupuesto de inotify, adopt, sentinel. |
| `mekami-cli/internal/watch` | sí | sí (1) | Filter, Coalescer, Translate, poller, paths, más una integración real de fsnotify. |
| `mekami-cli/tests/internal/install` | sí | — | Registro de cliente MCP black-box. |
| `mekami-cli/tests/cmd/mekami` | sí | — | Smoke black-box para el helper de truncado de `mcp-test`. |
| `mekami-core-go` | sí (2) | — | `imports_test.go` + `external_test/func_signature_test.go`. |
| `mekami-api` | — | — | Sin tests. |

## Convenciones

Estas son las reglas que sigue el suite; los tests nuevos deberían seguirlas también.

- **Solo `testing` estándar.** Usá `t.Errorf` / `t.Fatalf` para aserciones. No introduzcas `testify` ni `ginkgo`.
- **Subtests vía `t.Run`** para grupos de casos relacionados. Usá nombres de subtest en snake_case que se lean como un path (`ok`, `multiple_indexers_explicit_picks_requested`).
- **Table-driven cuando hay ≥3 casos similares.** Definí un slice `cases` de structs anónimos (o un map cuando el input es una key natural); cada caso lleva un `name` para el subtest.
- **Estado hermético.** Usá `t.TempDir()` para estado de filesystem, `t.Setenv()` para env, `t.Cleanup()` para todo lo demás. Nunca vayas a un `os.Setenv` / `os.Chdir` global directo.
- **`t.Helper()`** al tope de cada helper de test que llame a `t.Errorf` / `t.Fatalf`.
- **No `t.Parallel()`.** Los tests son rápidos y dependen de estado compartido en lugares (el registry `api.Global`, el state del supervisor). Agregar paralelismo es una decisión deliberada, no un default.
- **Skip, no fail, en gaps solo de entorno.** Usá `t.Skip("reason")` cuando el test no puede correr por un prerrequisito de plataforma faltante (no hay user bus de systemd, no hay `/proc`, etc.) y agregá un comentario explicando cómo habilitarlo.
- **`TestMain` es raro.** Solo dos archivos de test declaran uno: el registrador de stub-frontend en `ingest_test/setup_test.go` (build tag `!integration`) y el bootstrap vacío del integration-test en `integration_test/setup_test.go`.

## Helpers y stubs

Los helpers que se reutilizan entre paquetes viven en:

- `mekami-cli/internal/core/testutil/helpers.go` (paquete de producción, no `_test.go`). Expone `MustMkdir`, `MustWrite`, `WriteModuleFiles`, `OpenStoreForTest`, `QueriesStatsForTest`. Los tests black-box lo importan de la misma forma que el código de producción.
- `mekami-cli/internal/supervisor/testhelpers_test.go` y `mekami-cli/internal/watch/testhelpers_test.go` para helpers package-local (shim de fsnotify, daemons fake, servidores IPC stub, y un wrapper fino sobre `testutil.ShortSockDir` descripto abajo).
- `mekami-cli/internal/core/integration_test/bridge_test.go:buildTestGraph` es el helper canónico de "construir un grafo desde un blob de fuente Go" usado por la mayoría de los tests de integración.

Los tests que bindean un Unix socket deben usar `ShortSockDir(t)` desde `mekami-cli/internal/testutil/sockdir.go` (re-exportado como `shortSockDir(t)` desde los helpers de test por paquete) en lugar de `t.TempDir()` como parent del path del socket. En macOS el temp dir de runtime vive bajo `/var/folders/.../T/<name><digits>/<digits>/`, y una vez que le appendeás `.mekami/watcher.sock` el path completo excede el límite de 104 bytes de `sun_path` y `bind()` devuelve `invalid argument`. El helper es no-op en Linux/Windows (simplemente devuelve `t.TempDir()`) y en macOS parquea el dir bajo `/tmp/ms-<short-name>-XXXX` con un nombre truncado a 16 chars, así el path del socket queda bien por debajo del límite.

Hay tres stubs de `api.Frontend` en el suite:

- `mekami-cli/internal/core/ingest_test/stub_frontend_test.go` — stub completo backed por `go/parser` que devuelve el nombre de paquete y las declaraciones de primer nivel únicamente (sin imports, refs ni aristas de call). Registrado automáticamente en `TestMain` bajo el tag `!integration`.
- `mekami-cli/cmd/mekami/commands_test.go:fakeFrontend` — stub in-package mínimo para los tests de `resolveLang` / `resolveInitLangs` / `runInit`.
- `mekami-cli/internal/coreinstall/list_test.go:testFrontend` — stub mínimo para los tests de `List`.

Son intencionalmente chiquitos y no se consolidan — cada stub cubre solo la superficie que los tests de su paquete necesitan.

## CI y packaging

- **CI** (`.github/workflows/mekami.yml`): corre desde la raíz del repo así el `go.work` commiteado está en scope. El job `test` corre `go test -short ./...` contra el workspace (cubre `mekami-cli` y `mekami-core`) en Go 1.26 a través de `ubuntu-latest`, `macos-latest`, y `windows-latest`. Sin `-tags integration`, así que el suite de integración no se ejercita en CI. El job `build` corre `./build.sh` en Linux/macOS y un `go build ./...` plano en Windows.
- **AUR** (`.aur/mekami/PKGBUILD:check()`): corre desde la raíz del repo, llama a `go work sync` (idempotente; regenera el `go.work.sum` gitignored si falta) y luego `go test -short ./...`. La activación del workspace importa porque `mekami-cli/go.mod` requiere `mekami-core`, y en una build limpia de AUR ese módulo solo es resoluble como uno local a través del workspace — no desde el proxy de módulos.
- **Sin Makefile.** `build.sh` es un script de build solo de developer y no corre tests.

## Agregar un test nuevo

1. Elegí el paquete al que pertenece el test. Preferí `package <name>_test` (black-box) cuando el test ejercita la superficie pública; `package <name>` (white-box) solo cuando necesitás acceso a estado no exportado.
2. Si el test necesita un parser real de `mekami-core-go`, un watcher real de filesystem, o un user bus vivo, gatealo detrás de `//go:build integration`. Si depende de systemd de Linux, agregá `&& linux`.
3. Usá las convenciones de arriba: `t.TempDir()`, `t.Setenv()`, `t.Cleanup()`, `t.Helper()`, table-driven con `t.Run`.
4. Poné los helpers compartidos en `testutil/` (para cross-package) o `<pkg>/testhelpers_test.go` (para package-local).
5. Corré el suite localmente:
    ```bash
    go test ./...
    go test -tags integration ./...
    gofmt -l .   # tiene que estar vacío
    go vet ./...
    ```
6. CI no corre el suite de integración. Si tu cambio depende de que pasen los tests de integración, correlos localmente antes de abrir un PR.
