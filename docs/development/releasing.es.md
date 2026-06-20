# Releases

Mekami está dividido en tres repositorios. Los releases deben coordinarse para que el binario umbrella, el contrato de la API y cada core de lenguaje queden en lockstep.

## Reglas de SemVer

Taggeá los tres repos en lockstep. Un bump en la interfaz `api/v1` de `mekami-api` es un bump **major** para cada consumidor.

- **Major** (p. ej. `v1.0.0` → `v2.0.0`) — cualquier cambio breaking en `api/v1` o en la superficie de la CLI.
- **Minor** (p. ej. `v0.2.0` → `v0.3.0`) — nuevas features, nuevas herramientas, nuevos comandos, nuevas claves de config, todo backwards compatible.
- **Patch** (p. ej. `v0.2.3` → `v0.2.4`) — solo bug fixes, sin cambios de superficie.

## Procedimiento de bump

1. **Bumpeá la versión en el/los módulo(s) afectado(s):**
    - `mekami-cli` (binario, AUR): tag `v0.2.0` en la rama `main` del repo umbrella.
    - `mekami-core`: tag `v0.2.0` en su propio repo.
    - `mekami-core-<lang>`: tag `v0.2.0` en su propio repo.

2. **Bumpeá las líneas `require` en los `go.mod` downstream para matchear:**
    ```bash
    go get github.com/Wolf258/mekami-core@v0.2.0
    go get github.com/Wolf258/mekami-core-go@v0.2.0
    go mod tidy
    ```

3. **Commiteá las updates de `go.mod` / `go.sum` y pusheá.**

## Bump de AUR

Mirá el procedimiento completo en [Empaquetado AUR](../build/aur.md). La versión corta:

1. Tag upstream: `git tag v0.2.0 && git push origin v0.2.0`.
2. Subí los tarballs de release nombrados exactamente `mekami_0.2.0_linux_x86_64.tar.gz` y `mekami_0.2.0_linux_aarch64.tar.gz`.
3. Calculá el `sha256sum` de cada uno.
4. Actualizá `.aur/mekami-bin/PKGBUILD` (`pkgver` + checksums).
5. Actualizá `.aur/mekami/PKGBUILD` (`pkgver`).
6. Regenerá ambos archivos `.SRCINFO` con `makepkg --printsrcinfo > .SRCINFO`.
7. Commiteá + pusheá.
8. Pusheá a AUR.

## Version stamping

La versión se estampa en tiempo de build vía `-ldflags "-X ...install.version=..."`. La expresión `-ldflags` está inline en dos lugares, y deben mantenerse en lockstep:

- `build.sh` (builds de dev manuales)
- `.aur/mekami/PKGBUILD` (paquete AUR desde fuente)

Si el paquete de instalación alguna vez mueve la variable `version`, ambos archivos deben actualizarse juntos.

## Qué comunicar

El commit o la descripción de un PR de release debería llamar la atención sobre:

- Cuáles módulos se taggearon.
- Cualquier cambio en `api/v1` (y por lo tanto qué consumidores necesitan un bump major).
- Cualquier cambio en paquetes AUR (PKGBUILD, .SRCINFO).
- Cualquier cosa que requiera una acción del usuario (recompilar el binario, editar config, correr `core install`, etc.).
