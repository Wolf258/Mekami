# Empaquetado AUR

Mekami se distribuye por el [AUR](https://aur.archlinux.org). Los dos paquetes targetean el mismo binario `mekami`, pero lo traen de fuentes distintas.

| Paquete | Fuente | Para quién es |
| --- | --- | --- |
| `mekami-bin` | Tarball precompilado de release desde GitHub Releases. | Usuarios de Arch que no quieren un toolchain de Go. **Empezá por acá.** |
| `mekami` | Compila la CLI desde el tag de git upstream. | Usuarios que prefieren compilar desde fuente, o quieren los últimos cambios sin release. Requiere `go>=1.26`. |

Los dos paquetes entran en conflicto entre sí y ambos `provide` `mekami`, así que instalar uno automáticamente desinstala al otro. Esto matchea la convención usual de AUR para hermanos `-bin`.

## Instalar

```bash
yay -S mekami-bin    # binario precompilado
# o
yay -S mekami        # compila desde fuente
```

Verificá con `mekami --version`. La versión se estampa en tiempo de build vía `-ldflags "-X ...install.version=..."`.

## Bumpear un release

Los dos PKGBUILDs usan intencionalmente el mismo `pkgver` y el mismo tag upstream (`v<pkgver>`). El procedimiento de bump:

1. **Tag upstream.** Desde la raíz del repo:
    ```bash
    git tag v0.2.0
    git push origin v0.2.0
    ```

2. **Subí los tarballs de release.** El workflow de release debería publicar los assets con estos nombres exactos — el PKGBUILD los descarga por nombre, así que cualquier desvío rompe la build:
    ```text
    mekami_0.2.0_linux_x86_64.tar.gz
    mekami_0.2.0_linux_aarch64.tar.gz
    ```
    Cada tarball se expande en un binario `mekami` (y opcionalmente un archivo `LICENSE` en la raíz del archivo).

3. **Calculá los checksums.**
    ```bash
    sha256sum mekami_0.2.0_linux_x86_64.tar.gz \
              mekami_0.2.0_linux_aarch64.tar.gz
    ```

4. **Actualizá `.aur/mekami-bin/PKGBUILD`.**
    - Bumpeá `pkgver` a `0.2.0`.
    - Reemplazá los dos placeholders `sha256sums_*` con los valores del paso 3.

5. **Actualizá `.aur/mekami/PKGBUILD`.**
    - Bumpeá `pkgver` a `0.2.0` (la función `pkgver()` va a tomar el mismo valor del tag de git en tiempo de build; la línea literal solo se usa como fallback para la UI de `makepkg`).

6. **Regenerá ambos archivos `.SRCINFO`.** `makepkg` requiere esto y AUR va a rechazar submissions con entradas de SRCINFO stale.
    ```bash
    cd .aur/mekami     && makepkg --printsrcinfo > .SRCINFO
    cd .aur/mekami-bin && makepkg --printsrcinfo > .SRCINFO
    ```

7. **Commiteá + pusheá.** Commiteá `PKGBUILD` y `.SRCINFO` juntos; no toques los tarballs binarios desde este repo (viven en el release de GitHub, no acá).

8. **Pusheá a AUR.** Cloneá los repos del lado aur (uno por paquete) y pusheá el subárbol matcheante. El workflow estándar de AUR es:
    ```bash
    git clone ssh://aur@aur.archlinux.org/mekami-bin.git
    cp -r .aur/mekami-bin/* mekami-bin/
    cd mekami-bin && makepkg --printsrcinfo > .SRCINFO
    git add -A && git commit -m "mekami-bin: bump to 0.2.0" && git push
    ```

## Sanity check local

Antes de pushear a AUR, verificá que los PKGBUILDs compilen localmente:

```bash
# desde la raíz del repo
cd .aur/mekami-bin && makepkg -si    # precompilado, rápido
cd .aur/mekami     && makepkg -si    # desde fuente, más lento
```

`makepkg` reportará cualquier dependencia faltante, checksum roto, o error de sintaxis en el PKGBUILD mismo.

## ¿Por qué `build.sh` y el PKGBUILD comparten el mismo `-ldflags`?

La expresión `-ldflags` que estampa la versión en el binario vive en dos lugares:

- `build.sh` (builds de dev manuales — produce `./mekami` en la raíz del repo)
- `.aur/mekami/PKGBUILD` (paquete AUR desde fuente)

Ambos inlinean la expresión en vez de compartir un helper, así cada uno es un script autocontenido que el tooling de AUR puede parsear sin nuestro layout de repo. El instalador bootstrap (`scripts/install.sh`) se removió: los usuarios de AUR instalan vía `yay -S mekami-bin` (o `yay -S mekami`) y obtienen el binario en PATH directamente. **Si el paquete de instalación alguna vez mueve la variable `version`, ambos archivos deben actualizarse en lockstep.**
