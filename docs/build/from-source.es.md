# Compilar desde código fuente

`./build.sh` es un script de build solo de developer. **No** corre tests y **no** produce un paquete instalable; para eso, mirá [Empaquetado AUR](aur.md).

## Qué hace

```bash
./build.sh
```

1. Verifica que Go ≥ 1.26 esté instalado.
2. Regenera `mekami-cli/internal/core/frontend/all_gen/all_gen.go` con el conjunto dev (así las ediciones locales a `mekami-core-go` y amigos toman efecto).
3. Estampa la versión vía `-ldflags "-X ...install.version=..."`.
4. Produce `./mekami` en la raíz del repo.

El script es idempotente: re-correrlo es el comando canónico de "cambié un core, dame un binario fresco".

## Requisitos

- Go 1.26+.
- No se requiere un toolchain de C.
- `git` es necesario (el script `dev-allgen` lee tu cache de módulos).

## ¿Por qué `build.sh` y el PKGBUILD comparten el mismo `-ldflags`?

La expresión `-ldflags` que estampa la versión en el binario vive en dos lugares:

- `build.sh` (builds de dev manuales — produce `./mekami` en la raíz del repo)
- `.aur/mekami/PKGBUILD` (paquete AUR desde fuente)

Ambos inlinean la expresión en vez de compartir un helper, así cada uno es un script autocontenido que el tooling de AUR puede parsear sin nuestro layout de repo. El instalador bootstrap (`scripts/install.sh`) se removió: los usuarios de AUR instalan vía `yay -S mekami-bin` (o `yay -S mekami`) y obtienen el binario en PATH directamente.

**Si el paquete de instalación alguna vez mueve la variable `version`, ambos archivos deben actualizarse en lockstep.**

## Build manual (sin el script)

Si querés el build pelado, podés hacerlo a mano:

```bash
( cd mekami-cli && go build -o ../mekami ./cmd/mekami )
```

Vas a estar usando el `all_gen.go` de *producción* (el archivo tal como fue generado por última vez en el árbol de fuentes), no el conjunto dev. Usá `./build.sh` cuando hayas editado un core localmente.

## Windows

El script `build.sh` usa bash. En Windows, compilá con:

```powershell
cd mekami-cli
go build -o ..\mekami.exe .\cmd\mekami
```

El workflow de CI en `.github/workflows/mekami.yml` hace el equivalente en `windows-latest`.

## Cross-compilation

Mekami es un único binario Go. La cross-compilación es el baile estándar de `GOOS` / `GOARCH`:

```bash
GOOS=linux  GOARCH=amd64 go build -o mekami-linux-amd64   ./mekami-cli/cmd/mekami
GOOS=linux  GOARCH=arm64 go build -o mekami-linux-arm64   ./mekami-cli/cmd/mekami
GOOS=darwin GOARCH=arm64 go build -o mekami-darwin-arm64  ./mekami-cli/cmd/mekami
```

La convención de nombres de los tarballs de release usada por el paquete AUR `-bin` es `mekami_<pkgver>_linux_{x86_64,aarch64}.tar.gz`.

## Sanity check

Después de compilar:

```bash
./mekami --version
./mekami stats         # .mekami/config.json no existe todavía — esto solo imprime la versión
./mekami init          # en un repo de test, no en tu código real
./mekami build
./mekami find-symbol Foo
```
