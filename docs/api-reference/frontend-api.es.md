# Frontend API

El paquete `github.com/Wolf258/mekami-api/api/v1` es el contrato público que implementa cada indexer de lenguaje. El paquete solo depende de la stdlib de Go.

Esta página lista cada tipo y función exportada del paquete. Para las garantías del contrato y las formas de datos, mirá la página [Contrato de frontend](../extending/frontend-contract.md).

## Tipos

### `Workspace`

```go
type Workspace struct {
    IsWorkspace      bool
    WorkFile         string
    WorkspaceDir     string
    WorkspaceMods    []string
    PrimaryModPath   string
    PrimaryModuleDir string
}
```

### `FileMeta`

```go
type FileMeta struct {
    ModuleID  string
    PackageID string
    DirRel    string
}
```

### `ModuleInfo`

```go
type ModuleInfo struct {
    Dir      string
    ModuleID string
}
```

### `ParseResult`

```go
type ParseResult struct {
    RelPath    string
    Lang       string
    ModuleID   string
    PackageID  string
    DirRel     string
    Hash       string
    Mtime      int64
    Size       int64
    Symbols    []Symbol
    Refs       []Ref
}
```

### `Symbol`

```go
type Symbol struct {
    ID            int64
    FileID        int64
    PackageID     int64
    Kind          SymbolKind
    Name          string
    QualifiedName string
    StartLine     int
    EndLine       int
    Exported      bool
    Signature     string
    ParentSymbol  *int64
}
```

### `Ref`

```go
type Ref struct {
    ID          int64
    FromSymbol  int64
    ToQualified string
    Kind        RefKind
    Line        int
}
```

### `ModuleEntry`

```go
type ModuleEntry struct {
    Dir  string `json:"dir"`
    Path string `json:"path"`
}
```

### `Frontend` (interfaz)

```go
type Frontend interface {
    Name() string
    Extensions() []string
    ResolveLayout(root string) (*Workspace, error)
    ResolveModules(root string) ([]ModuleInfo, error)
    RootModule(root string) (string, error)
    ResolveFile(root, absPath string) (FileMeta, error)
    ParseFile(root, relPath, absPath string, hash string, mtime, size int64) (ParseResult, error)
    StructuralFiles() []string
    IsIndexable(relPath string) bool
}
```

Mirá la página [Contrato de frontend](../extending/frontend-contract.md) para la especificación completa método por método.

## Constantes

### `SymbolKind`

```go
const (
    KindFunc      SymbolKind = "func"
    KindMethod    SymbolKind = "method"
    KindType      SymbolKind = "type"
    KindVar       SymbolKind = "var"
    KindConst     SymbolKind = "const"
    KindImports   SymbolKind = "imports"
    KindFuncLit   SymbolKind = "funclit"
)
```

### `RefKind`

```go
const (
    RefCall    RefKind = "call"
    RefTypeUse RefKind = "type-use"
    RefImport  RefKind = "import"
    RefValue   RefKind = "value"
)
```

## Registry

### `Registry` y `Global`

```go
var Global = NewRegistry()

func NewRegistry() *Registry
func Register(f Frontend)               // atajo de Global.Register
func Get(name string) (Frontend, error) // atajo de Global.Get
func Names() []string                   // atajo de Global.Names
func All() []Frontend                   // atajo de Global.All
```

### Métodos de `Registry`

```go
func (r *Registry) Register(f Frontend)
func (r *Registry) Get(name string) (Frontend, error)
func (r *Registry) Names() []string
func (r *Registry) All() []Frontend
func (r *Registry) IsStructural(rel string) bool
func (r *Registry) DefaultStructuralFiles() []string
```

### Helpers a nivel de paquete

```go
func IsStructural(rel string) bool
func DefaultStructuralFiles() []string
```

`Register` hace panic con nombres duplicados, así un typo en un frontend se caza al arranque.

## Fuente

El fuente del paquete vive en el módulo `mekami-api`: <https://github.com/Wolf258/mekami-api/blob/main/api/v1/api.go>.
