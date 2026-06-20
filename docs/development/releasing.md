# Releasing

Mekami is split across three repositories. Releases must be coordinated so the umbrella binary, the API contract, and every language core stay in lockstep.

## SemVer rules

Tag all three repos in lockstep. A bump in `mekami-api`'s `api/v1` interface is a **major** bump for every consumer.

- **Major** (e.g. `v1.0.0` → `v2.0.0`) — any breaking change in `api/v1` or in the CLI surface.
- **Minor** (e.g. `v0.2.0` → `v0.3.0`) — new features, new tools, new commands, new config keys, all backwards compatible.
- **Patch** (e.g. `v0.2.3` → `v0.2.4`) — bug fixes only, no surface changes.

## Bump procedure

1. **Bump the version in the affected module(s):**
    - `mekami-cli` (binary, AUR): tag `v0.2.0` on the umbrella repo's `main` branch.
    - `mekami-core`: tag `v0.2.0` on its own repo.
    - `mekami-core-<lang>`: tag `v0.2.0` on its own repo.

2. **Bump the `require` lines in downstream `go.mod` files to match:**
    ```bash
    go get github.com/Wolf258/mekami-core@v0.2.0
    go get github.com/Wolf258/mekami-core-go@v0.2.0
    go mod tidy
    ```

3. **Commit the `go.mod` / `go.sum` updates and push.**

## AUR bump

See the full procedure at [AUR packaging](../build/aur.md). The short version:

1. Tag upstream: `git tag v0.2.0 && git push origin v0.2.0`.
2. Upload release tarballs named exactly `mekami_0.2.0_linux_x86_64.tar.gz` and `mekami_0.2.0_linux_aarch64.tar.gz`.
3. Compute `sha256sum` for each.
4. Update `.aur/mekami-bin/PKGBUILD` (`pkgver` + checksums).
5. Update `.aur/mekami/PKGBUILD` (`pkgver`).
6. Regenerate both `.SRCINFO` files with `makepkg --printsrcinfo > .SRCINFO`.
7. Commit + push.
8. Push to AUR.

## Version stamping

The version is stamped at build time via `-ldflags "-X ...install.version=..."`. The `-ldflags` expression is inlined in two places, and they must be kept in lockstep:

- `build.sh` (manual dev builds)
- `.aur/mekami/PKGBUILD` (AUR from-source package)

If the install package ever moves the `version` variable, both files must be updated together.

## What to communicate

A release commit or PR description should call out:

- Which modules were tagged.
- Any `api/v1` changes (and therefore which consumers need a major bump).
- Any AUR package changes (PKGBUILD, .SRCINFO).
- Anything that requires a user action (rebuild the binary, edit config, run `core install`, etc.).
