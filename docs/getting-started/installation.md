# Installation

Mekami ships through the AUR. There is no bootstrap installer — the AUR package places the binary at `/usr/bin/mekami` directly.

## Arch / Manjaro

```bash
yay -S mekami-bin    # prebuilt binary from GitHub Releases
# or
yay -S mekami        # builds from source (requires go >= 1.26)
```

The two packages `provide` and `conflict` with each other, so installing one removes the other. See [AUR packaging](../build/aur.md) for bump instructions.

Verify the result:

```bash
mekami --version
```

The version is stamped at build time via `-ldflags "-X ...install.version=..."`. Untouched builds report `dev`.

## Other distributions

Mekami is a single static Go binary with no CGo and no runtime dependencies beyond `glibc` (or `musl` for the AUR `-bin` build). To run it on any Linux, macOS, or Windows host:

1. Download the appropriate archive from the [GitHub Releases](https://github.com/Wolf258/mekami/releases) page.
2. Extract the binary and place it somewhere on your `$PATH`.
3. Verify with `mekami --version`.

## From source

If you want the latest unreleased code or you are contributing, build it locally:

```bash
git clone https://github.com/Wolf258/mekami
cd mekami
./build.sh
```

`./build.sh` checks that Go ≥ 1.26 is installed, regenerates `all_gen.go` with the dev builtin set, stamps the version via `-ldflags`, and produces `./mekami`. See [Build & install](../build/from-source.md) for the full mechanics.

## Wire Mekami into an MCP client

Once the binary is on your `$PATH`, register it with your MCP-aware client (OpenCode, Claude Desktop, etc.):

```bash
mekami mcp install
```

This writes an `mcp.mekami` entry into the user's `opencode.json` (respecting `$XDG_CONFIG_HOME`) with the portable form:

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

The original file is backed up to `opencode.json.bak` before any change.

Useful flags:

| Flag | Effect |
| --- | --- |
| `--binary /abs/path/mekami` | Pin the entry to a specific binary (useful for dev builds). |
| `--name <other>` | Register under a different server name. |
| `--disable` | Register with `enabled: false` (turn it on later from the client UI). |
| `--env KEY=VALUE` | Inject an environment variable into the spawned `mekami serve` process. Repeatable. |

To remove the entry:

```bash
mekami mcp uninstall
```

You can also use `--name` to remove a custom-registered entry.

## Next step

Proceed to the [quick start](quickstart.md) to build your first index and ask your first structural question.
