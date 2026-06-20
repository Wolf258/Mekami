# Contribuir

## Estilo de código

- `gofmt` y `go vet` son la fuente de verdad. Corré ambos antes de abrir un PR.
- Sin frameworks externos de test. Solo `testing` estándar.
- Sin CGo. Usá `modernc.org/sqlite` y la stdlib pura de Go.
- Sin comentarios a menos que expliquen el *por qué*. El código debería hablar por sí mismo; reservá los comentarios para invariantes, ordenamientos sorprendentes y referencias a issues upstream.

## El workflow de regeneración de `all_gen`

`mekami-cli/internal/core/frontend/all_gen/all_gen.go` está generado. No lo edites a mano.

Cuando hay que regenerar el archivo, `./build.sh` lo hace por vos. Si querés correr el script aislado:

```bash
( cd mekami-cli && go run ./internal/core/scripts/dev-allgen )
```

El script es idempotente: siempre produce el set completo, nunca un delta. Si el diff es inesperado, revisá `go.work` y tu cache de módulos.

Mirá [El mecanismo `all_gen`](../extending/all-gen.md) para la historia completa dev-vs-prod.

## Version stamping

La versión se estampa en tiempo de build vía `-ldflags "-X ...install.version=..."`. Las builds sin tocar reportan `dev`.

La expresión `-ldflags` está inline en dos lugares, y deben mantenerse en lockstep:

- `build.sh` (builds de dev manuales — produce `./mekami` en la raíz del repo)
- `.aur/mekami/PKGBUILD` (paquete AUR desde fuente)

Si el paquete de instalación alguna vez mueve la variable `version`, ambos archivos deben actualizarse juntos.

## Proceso de pull request

1. **Abrí un issue primero** para cualquier cambio no trivial. Mekami todavía está en etapa temprana, y una conversación rápida de diseño al arranque le ahorra tiempo a todos.
2. **Mantené el PR enfocado.** Un cambio, un PR. Separá los refactors de los cambios de features.
3. **Corré el suite de tests localmente** antes de pushear:
    ```bash
    go test ./...
    go test -tags integration ./...
    gofmt -l .
    go vet ./...
    ```
4. **Actualizá la docs** si el cambio toca la superficie para el usuario (CLI, herramientas MCP, configuración, modo watch). Las docs viven bajo `docs/` y se mantienen en sync entre `en` y `es`.
5. **No commitees secretos.** Sin API keys, sin tokens, sin paths de `home/`.

## Agregar un nuevo core de lenguaje

Mirá el walkthrough completo en [Escribir un frontend](../extending/writing-a-frontend.md). La versión corta:

1. Creá el repo en `github.com/Wolf258/mekami-core-<lang>`.
2. Inicializá el módulo Go y traete `mekami-api`.
3. Implementá `api.Frontend`.
4. Auto-registrate en `init()`.
5. Taggeá el primer release.
6. Desde el árbol de fuentes de mekami: `mekami core install <lang>@v0.1.0`.
7. Rebuildeá y verificá.

## Agregar un test nuevo

Mirá [Testing](testing.md#agregar-un-test-nuevo) para el checklist completo. La versión corta:

1. Elegí el paquete. Preferí `package <name>_test` (black-box).
2. Usá el build tag `integration` si el test necesita un frontend real, un watcher real, o un user bus vivo.
3. Seguí las convenciones: `t.TempDir()`, `t.Setenv()`, `t.Cleanup()`, `t.Helper()`, table-driven, no `t.Parallel()`.
4. Poné los helpers compartidos en `testutil/` o `<pkg>/testhelpers_test.go`.
5. Corré el suite localmente antes de pushear.

## Reportar bugs

El directorio `eval/` es la casa para los reportes de triage de issues. Hoy solo tiene un placeholder. Una vez que se reproduce un bug, el reporte va ahí como `eval/<date>-<short-slug>.md`.
