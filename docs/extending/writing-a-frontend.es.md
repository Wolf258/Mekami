# Escribir un frontend

Este walkthrough agrega un indexer hipotético `mylang`. La forma de un frontend son unas 100-300 líneas más el parser. La estrategia recomendada para el parser es vincular a [tree-sitter](https://tree-sitter.github.io/tree-sitter/) (un único binding Go libre de CGo maneja todas las gramáticas).

## 1. Crear un nuevo módulo

```text
mekami-core-mylang/
    go.mod
    frontend.go     # implementación de Frontend
    parser.go       # el pegamento de tree-sitter
    helpers.go      # extracción de símbolos, colección de refs
```

El `go.mod` debería requerir solo `github.com/Wolf258/mekami-api` y (para workspaces estilo Go) `github.com/Wolf258/mekami-core-go/modlayout` — los lenguajes no Go pueden omitir el último.

```bash
mkdir mekami-core-mylang && cd mekami-core-mylang
go mod init github.com/Wolf258/mekami-core-mylang
go get github.com/Wolf258/mekami-api@v0.1.0
```

## 2. Implementar la interfaz `api.Frontend`

```go
package mylang

import (
    "github.com/Wolf258/mekami-api/api/v1"
)

type Frontend struct{}

func (Frontend) Name() string                          { return "mylang" }
func (Frontend) Extensions() []string                  { return []string{".ml"} }
func (Frontend) StructuralFiles() []string             { return []string{"mylang.toml"} }
func (Frontend) IsIndexable(rel string) bool           { return true }
func (Frontend) ResolveLayout(root string) (*api.Workspace, error) {
    return &api.Workspace{}, nil
}
func (Frontend) ResolveModules(root string) ([]api.ModuleInfo, error) {
    return []api.ModuleInfo{{Dir: root, ModuleID: "mylang-root"}}, nil
}
func (Frontend) RootModule(root string) (string, error) { return "mylang-root", nil }
func (Frontend) ResolveFile(root, abs string) (api.FileMeta, error) {
    // Buscar los identifiers de proyecto / paquete para abs.
}
func (Frontend) ParseFile(root, rel, abs string, hash string, mtime, size int64) (api.ParseResult, error) {
    // Leer abs, parsearlo, devolver symbols + refs.
    // `Refs[i].FromSymbol` es el índice 0-based en el slice `Symbols`
    // devuelto; el writer lo resuelve a un id real.
}
```

Mirá el [contrato de frontend](frontend-contract.md) para la especificación método por método.

## 3. Auto-registrarse en `init()`

```go
func init() { api.Register(Frontend{}) }
```

El registry `api.Global` hace panic con nombres duplicados, así un typo en un frontend se caza al arranque.

## 4. Taggear el primer release

```bash
git tag v0.1.0
git push origin main v0.1.0
```

## 5. Registrar el indexer para un proyecto

Desde adentro del árbol de fuentes de mekami (donde vive `go.work`):

```bash
mekami core install mylang@v0.1.0
```

Esto escribe `{ "mylang": "v0.1.0" }` a los `indexers` de `.mekami/config.json`, regenera `mekami-cli/internal/core/frontend/all_gen/all_gen.go` con un blank import nuevo, e imprime un hint de rebuildear el binario.

En producción (instalación de AUR), el binario es read-only y el usuario necesita actualizar el paquete para tomar los cores recién instalados. En dev, corré `./build.sh` para recompilar con el nuevo blank import.

## 6. Verificar

```bash
./build.sh
./mekami core list        # debería mostrar "mylang" ahora
./mekami build --lang mylang
./mekami find-symbol Foo
```

Los comandos `core list` y `core status` son tu primer sanity check. `core status` reporta los frontends que están listados en la config pero cuyo blank import falta como `missing`.

## Walkthrough concreto: `mekami-core-rust`

Suponé que estás agregando `mekami-core-rust`.

1. **Creá el repo:**
    ```bash
    gh repo create Wolf258/mekami-core-rust --public
    ```

2. **Adentro del nuevo repo, iniciá el módulo Go y traete `mekami-api`:**
    ```bash
    go mod init github.com/Wolf258/mekami-core-rust
    go get github.com/Wolf258/mekami-api@v0.1.0
    ```

3. **Implementá `api.Frontend` desde `github.com/Wolf258/mekami-api/api/v1`.** La interfaz es chiquita — mirá `mekami-core-go` (`parser.go`) para una implementación de referencia. Cada método tiene un docstring que explica el contrato.

4. **Agregá un blank import en el archivo de entry del core para que se auto-registre vía `init()`:**
    ```go
    package rustfrontend

    import _ "github.com/Wolf258/mekami-api/api/v1"

    func init() { v1.Register(Frontend{}) }
    ```

5. **Taggeá el primer release:**
    ```bash
    git tag v0.1.0
    git push origin main v0.1.0
    ```

6. **En el repo umbrella `Mekami`, `coreinstall` lo levanta automáticamente** — `ModulePath("rust")` devuelve `github.com/Wolf258/mekami-core-rust` y el resolver lo trae del proxy por versión. No se necesita ningún cambio de código en `mekami-cli/internal/coreinstall/lang.go`.

7. **Probá:**
    ```bash
    go test ./...
    ./build.sh
    ./mekami core install rust
    ./mekami core list    # debería mostrar "rust" ahora
    ```

## Errores comunes

- **Olvidarse de rebuildear el binario.** El manifiesto de blank import se lee en tiempo de compilación. Un `core install` nuevo no toma efecto hasta que `./build.sh` (o un paquete AUR nuevo) se corra.
- **Devolver `nil` para `Symbols` o `Refs`.** El writer los indexa directamente; `nil` va a panickear durante el bulk insert. Devolvé siempre `make([]X, 0)` cuando el archivo no tiene entradas.
- **Qualified names no deterministas.** El frontend Go deriva `qualified_name` del package path del archivo + el nombre local del símbolo. Si tus nombres no son estables entre rebuilds, `find_symbol` devolverá hits duplicados para la misma declaración.
- **I/O bloqueante adentro de `ParseFile`.** El pipeline corre `runtime.NumCPU()` workers. Un frontend lento estanca todo el pool. Parseá un archivo, no llames a recursos de red, y dejá que la cola del pool de workers absorba el resto.
