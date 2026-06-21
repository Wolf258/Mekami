package format

import (
	"strings"
	"testing"

	"github.com/Wolf258/mekami-cli/internal/core/model"
)

func TestMaybeHeader_NotTruncated(t *testing.T) {
	if got := MaybeHeader(KindRefs, Cap{Total: 3, Shown: 3}); got != "" {
		t.Fatalf("expected empty header when not truncated, got %q", got)
	}
}

func TestMaybeHeader_Singular(t *testing.T) {
	got := MaybeHeader(KindRefs, Cap{Total: 1, Shown: 1, Truncated: true})
	if !strings.Contains(got, "1 reference found") {
		t.Fatalf("expected singular noun, got %q", got)
	}
}

func TestMaybeHeader_Plural(t *testing.T) {
	got := MaybeHeader(KindRefs, Cap{Total: 130, Shown: 30, Truncated: true})
	if !strings.Contains(got, "130 references found") {
		t.Fatalf("expected plural noun with count, got %q", got)
	}
	if !strings.Contains(got, "showing first 30 of 130") {
		t.Fatalf("expected shown/total in header, got %q", got)
	}
}

func TestMaybeFooter_Empty(t *testing.T) {
	if got := MaybeFooter(Cap{Truncated: false, Hint: "x"}); got != "" {
		t.Fatalf("expected empty footer when not truncated, got %q", got)
	}
	if got := MaybeFooter(Cap{Truncated: true, Hint: ""}); got != "" {
		t.Fatalf("expected empty footer when hint empty, got %q", got)
	}
}

func TestMaybeFooter_Indented(t *testing.T) {
	got := MaybeFooter(Cap{Truncated: true, Hint: "tip: narrow it"})
	if !strings.HasPrefix(got, "  ") {
		t.Fatalf("expected leading indent, got %q", got)
	}
	if !strings.Contains(got, "tip: narrow it") {
		t.Fatalf("expected hint text, got %q", got)
	}
}

func TestHintFor_KnownKinds(t *testing.T) {
	for _, k := range []ListKind{KindRefs, KindSymbols, KindMatches, KindFiles, KindModules, KindPackages, KindImporters, KindChanges, KindOutgoing, KindSites} {
		if HintFor(k) == "" {
			t.Fatalf("HintFor(%q) returned empty", k)
		}
	}
}

func TestRefsTo_NotTruncated(t *testing.T) {
	refs := []model.RefSite{
		{FromSymbol: model.SymbolWithFile{Symbol: model.Symbol{QualifiedName: "a.X"}, FilePath: "a.go"}, Line: 1, Kind: "call"},
		{FromSymbol: model.SymbolWithFile{Symbol: model.Symbol{QualifiedName: "b.Y"}, FilePath: "b.go"}, Line: 2, Kind: "call"},
	}
	got := RefsTo("target", refs, Cap{Total: 2, Shown: 2})
	if strings.HasPrefix(got, "2 references found") {
		t.Fatalf("non-truncated should not emit header, got %q", got)
	}
	if !strings.Contains(got, "references to \"target\"") {
		t.Fatalf("missing main header, got %q", got)
	}
}

func TestRefsTo_Truncated(t *testing.T) {
	refs := []model.RefSite{
		{FromSymbol: model.SymbolWithFile{Symbol: model.Symbol{QualifiedName: "a.X"}, FilePath: "a.go"}, Line: 1, Kind: "call"},
		{FromSymbol: model.SymbolWithFile{Symbol: model.Symbol{QualifiedName: "b.Y"}, FilePath: "b.go"}, Line: 2, Kind: "call"},
	}
	got := RefsTo("target", refs, Cap{Total: 130, Shown: 2, Truncated: true, Hint: HintFor(KindRefs)})
	if !strings.HasPrefix(got, "130 references found") {
		t.Fatalf("truncated should emit header first, got %q", got)
	}
	if !strings.Contains(got, "tip:") {
		t.Fatalf("expected footer tip, got %q", got)
	}
	// Only 2 ref lines should be present.
	if strings.Count(got, "\n  ") < 2 {
		t.Fatalf("expected at least 2 ref lines, got %q", got)
	}
}

func TestRefsTo_TruncationDropsItems(t *testing.T) {
	// Pass 5 refs but cap to 2; output should not contain the 3rd-5th.
	refs := make([]model.RefSite, 5)
	for i := range refs {
		refs[i] = model.RefSite{FromSymbol: model.SymbolWithFile{Symbol: model.Symbol{QualifiedName: "p.Q"}, FilePath: "p.go"}, Line: i + 1, Kind: "call"}
	}
	got := RefsTo("t", refs, Cap{Total: 5, Shown: 2, Truncated: true, Hint: "x"})
	// The 3rd-5th lines (with their unique line numbers) must be absent.
	for _, ln := range []string{":3 ", ":4 ", ":5 "} {
		if strings.Contains(got, ln) {
			t.Fatalf("output contains line %q which should have been dropped: %q", ln, got)
		}
	}
}

func TestRefsFrom_Truncated(t *testing.T) {
	qns := []string{"a.B", "a.C", "a.D", "a.E", "a.F"}
	got := RefsFrom("src", qns, Cap{Total: 50, Shown: 5, Truncated: true, Hint: HintFor(KindOutgoing)})
	if !strings.HasPrefix(got, "50 outgoing references found") {
		t.Fatalf("expected outgoing header, got %q", got)
	}
	for _, q := range qns {
		if !strings.Contains(got, q) {
			t.Fatalf("expected %q in output, got %q", q, got)
		}
	}
}

func TestFileOutline_Empty(t *testing.T) {
	got := FileOutline(nil, Cap{Total: 0, Shown: 0})
	if got != "(no symbols)" {
		t.Fatalf("expected (no symbols), got %q", got)
	}
}

func TestFileOutline_NotTruncated(t *testing.T) {
	syms := []model.SymbolWithFile{
		{Symbol: model.Symbol{QualifiedName: "a.X", StartLine: 1, EndLine: 5, Kind: "func"}, FilePath: "a.go"},
	}
	got := FileOutline(syms, Cap{Total: 1, Shown: 1})
	if strings.HasPrefix(got, "1 symbol found") {
		t.Fatalf("non-truncated should not emit header, got %q", got)
	}
	if !strings.Contains(got, "a.go") {
		t.Fatalf("expected file path, got %q", got)
	}
}

func TestFileOutline_Truncated(t *testing.T) {
	syms := []model.SymbolWithFile{
		{Symbol: model.Symbol{QualifiedName: "a.X", StartLine: 1, EndLine: 5, Kind: "func"}, FilePath: "a.go"},
		{Symbol: model.Symbol{QualifiedName: "a.Y", StartLine: 10, EndLine: 15, Kind: "func"}, FilePath: "a.go"},
		{Symbol: model.Symbol{QualifiedName: "a.Z", StartLine: 20, EndLine: 25, Kind: "func"}, FilePath: "a.go"},
	}
	got := FileOutline(syms, Cap{Total: 100, Shown: 2, Truncated: true, Hint: HintFor(KindSymbols)})
	if !strings.HasPrefix(got, "100 symbols found") {
		t.Fatalf("expected header, got %q", got)
	}
	if !strings.Contains(got, "tip:") {
		t.Fatalf("expected footer, got %q", got)
	}
	// The 3rd symbol (a.Z) should be absent.
	if strings.Contains(got, "a.Z") {
		t.Fatalf("truncated output should drop a.Z, got %q", got)
	}
}

func TestPackageOutline_HeaderHasCapTotal(t *testing.T) {
	syms := []model.SymbolWithFile{
		{Symbol: model.Symbol{QualifiedName: "a.X", StartLine: 1, EndLine: 5, Kind: "func"}, FilePath: "a.go"},
	}
	got := PackageOutline("github.com/foo/bar", syms, Cap{Total: 50, Shown: 1, Truncated: true, Hint: HintFor(KindSymbols)})
	if !strings.HasPrefix(got, "50 symbols found") {
		t.Fatalf("expected package header to lead with cap header, got %q", got)
	}
	if !strings.Contains(got, "package github.com/foo/bar") {
		t.Fatalf("expected package line, got %q", got)
	}
}

func TestModuleOverview_Truncated(t *testing.T) {
	mods := []model.ModuleSummary{
		{ModuleID: "github.com/foo/a", Packages: []model.PackageSummary{{PackageID: "github.com/foo/a/p", Files: 1, Symbols: 2, Exported: 1}}},
		{ModuleID: "github.com/foo/b", Packages: []model.PackageSummary{{PackageID: "github.com/foo/b/p", Files: 1, Symbols: 2, Exported: 1}}},
	}
	got := ModuleOverview(mods, Cap{Total: 20, Shown: 2, Truncated: true, Hint: HintFor(KindModules)})
	if !strings.HasPrefix(got, "20 modules found") {
		t.Fatalf("expected header, got %q", got)
	}
	if !strings.Contains(got, "tip:") {
		t.Fatalf("expected footer, got %q", got)
	}
}

func TestModuleOverview_Empty(t *testing.T) {
	got := ModuleOverview(nil, Cap{Total: 0, Shown: 0})
	if got != "(no modules)" {
		t.Fatalf("expected (no modules), got %q", got)
	}
}
