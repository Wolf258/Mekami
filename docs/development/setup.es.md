# Setup

## Prerrequisitos

- **Go 1.26+** (matchea la versión en `mekami-cli/go.mod`).
- **git**.
- **sqlite3** CLI opcional, solo para hurgar a mano en los archivos `.mekami/*.db`.
- No se requiere un toolchain de C — Mekami usa `modernc.org/sqlite` (Go puro) y solo el toolchain de Go.

## Layout del repositorio

Mekami está dividido en tres repositorios públicos para que cada componente externo se pueda consumir, versionar y testear independientemente. El pipeline de indexación que solía vivir en un repo separado `mekami-core` ahora está fusionado en el umbrella como `mekami-cli/internal/core/`:

```text
Wolf258/mekami-api         ← api/v1/ (el contrato de la interfaz Frontend)
Wolf258/Mekami             ← umbrella: mekami-cli (con internal/core) + go.work
Wolf258/mekami-core-go     ← frontend del lenguaje Go
```

El repo umbrella `Mekami` contiene todo el binario como un único módulo Go en `mekami-cli/`, con el árbol del antiguo `mekami-core` viviendo bajo `internal/core/`. Un archivo `go.work` commiteado en la raíz del repo apunta a `mekami-cli` para que los comandos de build desde la raíz sigan andando. La CLI blank-importa `mekami-core-go` desde el `all_gen.go` generado para registrar el frontend Go en `api.Global`.

`mekami-api` y `mekami-core-go` quedan como repositorios externos. Se los trae del proxy de módulos de Go por versión.

Todos los módulos se publican bajo `github.com/Wolf258/...`. El prefijo `mekami/...` no se usa porque el org de GitHub de ese nombre es de otro.

### Qué vive dónde

- **`mekami-api`** — stdlib puro, sin dependencias internas. Solo la interfaz `api.Frontend` y las formas de datos compartidas (`ParseResult`, `Symbol`, `Ref`, `Workspace`, `ModuleInfo`, `ModuleEntry`). Bumpear esto es un major version para cada consumidor downstream.
- **`mekami-core`** — pipeline de indexación agnóstico del lenguaje: ingest, store, queries, walker, diff, grep. Importa `mekami-api` por el contrato. **No** conoce a Go, Rust, etc. directamente. Su única asunción específica del lenguaje es que cualquier frontend puede responder `ResolveLayout`, `ResolveModules`, `RootModule`, `ResolveFile`, `ParseFile`.
- **`mekami-cli`** — el binario. Importa `mekami-core` y blank-importa los cores de lenguaje que el usuario instaló (`core install go` etc.).
- **`mekami-core-go`** — el frontend de Go. Implementa `api.Frontend` y se autoregistra en `init()`. Importa `mekami-api` por el contrato; **no** importa `mekami-core` (que mantiene el grafo de módulos acíclico).

## Setup básico

Esto es lo que haría un contribuidor que solo quiere arreglar un bug de la CLI. No se necesita setup de core de lenguaje ni de dev de core.

```bash
git clone https://github.com/Wolf258/Mekami
cd Mekami
go version                      # tiene que ser 1.26+

# Testear todo en el workspace (cli + core).
go test ./...

# Compilar el binario.
./build.sh
./mekami --version
```

El `go.work` commiteado en la raíz del repo trae `./mekami-cli`, así que `go test ./mekami-cli/...` desde la raíz cubre el binario entero. No se necesita setup manual de workspace para el caso común.

`./build.sh` corre el script dev-allgen, regenera `mekami-cli/internal/core/frontend/all_gen/all_gen.go` con los cores que estén resolubles, y produce un binario `mekami` en la raíz del repo.

La CLI depende de `github.com/Wolf258/mekami-core`, `github.com/Wolf258/mekami-api`, y `github.com/Wolf258/mekami-core-go` (vía `go.mod`). Los tres se traen del proxy de Go por versión. No se requiere directiva `replace`.

## Dev local con múltiples módulos

Si querés desarrollar `mekami-cli` junto con ediciones locales a `mekami-api` o `mekami-core-go` para que esas tomen efecto sin publicar un tag, reemplazá el `require` correspondiente en `mekami-cli/go.mod` con una directiva `replace ... => ../<sibling>`, y luego corré `go mod tidy`. El `go.work` commiteado en este repo ya no se usa para eso — el binario es un módulo ahora.

### Comandos útiles de `go work`

```bash
# Agregar un nuevo módulo local al workspace.
go work use ../mekami-core-rust

# Mostrar la definición actual del workspace.
go work edit -print

# Sincronizar el workspace tras editar go.mod files.
go work sync

# Quitar un módulo del workspace.
go work edit -dropreplace=../mekami-core-rust
```

## Comandos comunes

```bash
# Correr todos los tests a través del workspace (usa el go.work commiteado).
go test ./...

# Testear un único módulo.
( cd mekami-cli && go test ./... )

# Correr solo los tests matcheados, p. ej. supervisor.
( cd mekami-cli && go test ./internal/supervisor/... )

# Regenerar el manifiesto de blank imports all_gen.go.
( cd mekami-cli && go run ./internal/core/scripts/dev-allgen )

# Compilar el binario de la CLI.
./build.sh

# Re-compilar tras cambiar un core.
./build.sh && ./mekami core list
```

## Troubleshooting

### `pattern ./... matches no packages`

Estás corriendo `go test ./...` desde un directorio que no tiene `go.mod` y no es parte del workspace. Asegurate de estar en la raíz del repo (donde vive `go.work`) y que el archivo esté intacto. Para correr un único módulo aislado:

```bash
( cd mekami-cli && go test ./... )
```

### Rompiste el workspace sin querer

El `go.work` commiteado lista solo `./mekami-cli` y está pensado para trackearse. Si lo editaste (o generaste un `go.work.sum` contra un layout solo-local) y querés restaurar el contenido commiteado:

```bash
git checkout -- go.work
rm -f go.work.sum
```

Si estás apuntando el workspace a clones locales de `mekami-api` o `mekami-core-go` para laburo e2e, preferí una directiva `replace` en `mekami-cli/go.mod` a mutar `go.work`, así el cambio queda local a tu checkout y desaparece con el working tree.
