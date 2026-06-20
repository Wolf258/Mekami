# Instalación

Mekami se distribuye por AUR. No hay instalador bootstrap — el paquete de AUR coloca el binario directamente en `/usr/bin/mekami`.

## Arch / Manjaro

```bash
yay -S mekami-bin    # binario precompilado desde GitHub Releases
# o
yay -S mekami        # compila desde el código fuente (requiere go >= 1.26)
```

Los dos paquetes se `provide` y `conflict` mutuamente, así que instalar uno desinstala el otro. Mirá [Empaquetado AUR](../build/aur.md) para el procedimiento de bump.

Verificá el resultado:

```bash
mekami --version
```

La versión se estampa en tiempo de compilación vía `-ldflags "-X ...install.version=..."`. Las builds sin tocar reportan `dev`.

## Otras distribuciones

Mekami es un único binario estático de Go sin CGo y sin dependencias de runtime más allá de `glibc` (o `musl` para la build `-bin` de AUR). Para correrlo en cualquier host Linux, macOS o Windows:

1. Descargá el archivo apropiado desde la página de [GitHub Releases](https://github.com/Wolf258/mekami/releases).
2. Extraé el binario y ponelo en tu `$PATH`.
3. Verificá con `mekami --version`.

## Desde el código fuente

Si querés lo último sin release o estás contribuyendo, compilá localmente:

```bash
git clone https://github.com/Wolf258/mekami
cd mekami
./build.sh
```

`./build.sh` verifica que Go ≥ 1.26 esté instalado, regenera `all_gen.go` con el conjunto dev, estampa la versión vía `-ldflags` y produce `./mekami`. Mirá [Build e instalación](../build/from-source.md) para la mecánica completa.

## Conectar Mekami a un cliente MCP

Una vez que el binario está en tu `$PATH`, registralo con tu cliente compatible con MCP (OpenCode, Claude Desktop, etc.):

```bash
mekami mcp install
```

Esto escribe una entrada `mcp.mekami` en el `opencode.json` del usuario (respetando `$XDG_CONFIG_HOME`) con la forma portable:

```json
{
  "mcp": {
    "mekami": {
      "type": "local",
      "command": ["mekami", "serve"],
      "enabled": true
    }
  }
}
```

El archivo original se respalda a `opencode.json.bak` antes de cualquier cambio.

Flags útiles:

| Flag | Efecto |
| --- | --- |
| `--binary /abs/path/mekami` | Pinea la entrada a un binario específico (útil para builds de dev). |
| `--name <otro>` | Registra bajo otro nombre de servidor. |
| `--disable` | Registra con `enabled: false` (lo activás después desde el cliente). |
| `--env KEY=VALUE` | Inyecta una variable de entorno al proceso `mekami serve` que se lanza. Repetible. |

Para eliminar la entrada:

```bash
mekami mcp uninstall
```

También podés usar `--name` para eliminar una entrada registrada con nombre custom.

## Siguiente paso

Continuá con el [inicio rápido](quickstart.md) para construir tu primer índice y hacer tu primera pregunta estructural.
