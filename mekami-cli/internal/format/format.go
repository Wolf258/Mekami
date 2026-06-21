package format

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Wolf258/mekami-cli/internal/core/model"
)

// JSON encodes v as an indented JSON string. If v is already a string
// (typical for human-readable formatters like format.Symbol), it is
// returned verbatim. Any encoding error is returned as a string
// instead of an error so callers can pass the result to wire formats
// (CLI stdout, MCP TextContent) without losing the payload.
func JSON(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("format.JSON: marshal failed: %v", err)
	}
	return string(out)
}

// Cap is the truncation metadata emitted alongside any list-shaped
// formatter when the result was longer than the visible cap. The
// fields are populated only when there is something to report; the
// JSON tag omitempty keeps short responses identical to the
// pre-cap shape.
type Cap struct {
	// Total is the number of items the underlying query produced
	// before the cap was applied. When Truncated is false, Total
	// equals Shown.
	Total int `json:"total,omitempty"`
	// Shown is the number of items actually included in the output.
	Shown int `json:"shown,omitempty"`
	// Truncated is true when Shown < Total. Consumers can use it as
	// a fast-path "was the cap hit" check.
	Truncated bool `json:"truncated,omitempty"`
	// Hint is a one-line suggestion telling the caller how to
	// re-narrow the query (e.g. "use --ref-kind=call or
	// --path-prefix=<subdir>"). Empty when the result was not
	// truncated.
	Hint string `json:"hint,omitempty"`
}

// ListKind is a small enum of "what kind of list is this" so the
// header/footer copy can mention the right noun without each
// formatter having to hardcode it.
type ListKind string

const (
	KindRefs     ListKind = "references"
	KindSymbols  ListKind = "symbols"
	KindMatches  ListKind = "matches"
	KindFiles    ListKind = "files"
	KindModules  ListKind = "modules"
	KindPackages ListKind = "packages"
	KindImporters ListKind = "importers"
	KindChanges  ListKind = "changes"
	KindSites    ListKind = "sites"
	KindOutgoing ListKind = "outgoing references"
)

// headerNoun returns the singular/plural noun used in the header
// "N noun found" line.
func headerNoun(k ListKind) string {
	switch k {
	case KindRefs:
		return "reference"
	case KindSymbols:
		return "symbol"
	case KindMatches:
		return "match"
	case KindFiles:
		return "file"
	case KindModules:
		return "module"
	case KindPackages:
		return "package"
	case KindImporters:
		return "importer"
	case KindChanges:
		return "change"
	case KindSites:
		return "site"
	case KindOutgoing:
		return "outgoing reference"
	}
	return "item"
}

// HintFor returns the user-facing hint string for a given list kind.
// It is the footer copy printed (and JSON-serialized) when the
// output was truncated. Empty for kinds that do not have a useful
// narrowing suggestion.
func HintFor(k ListKind) string {
	switch k {
	case KindRefs, KindSites:
		return "tip: re-run with --ref-kind=<call|type-use|value|import> or --path-prefix=<subdir> to narrow the result."
	case KindSymbols:
		return "tip: re-run with --kind=<func|type|var|const> or --path-prefix=<subdir> to narrow the result."
	case KindMatches:
		return "tip: re-run with --path-prefix=<subdir> or --include-ext=<go,md> to narrow the result."
	case KindFiles:
		return "tip: re-run with --prefix=<subdir> or --include=<go,md> to narrow the result."
	case KindModules:
		return "tip: this list is exhaustive by design; --head 0 disables the cap."
	case KindPackages:
		return "tip: re-run with --kinds=<func,type> to narrow the symbol set, or pass the canonical package_id."
	case KindImporters:
		return "tip: pass the canonical import path (not the bare last segment) to disambiguate."
	case KindChanges:
		return "tip: re-run `mekami build` to refresh the index, then re-query."
	case KindOutgoing:
		return "tip: re-run with --path-prefix=<subdir> to narrow the result."
	}
	return ""
}

// MaybeHeader returns the "N references found — showing first M of N"
// line when cap.Truncated is true, else "". It is intended to be
// prepended to the formatted list. Pluralization is automatic.
func MaybeHeader(k ListKind, cap Cap) string {
	if !cap.Truncated || cap.Total <= 0 {
		return ""
	}
	noun := headerNoun(k)
	if cap.Total == 1 {
		return fmt.Sprintf("1 %s found.\n", noun)
	}
	return fmt.Sprintf("%d %ss found — showing first %d of %d.\n",
		cap.Total, noun, cap.Shown, cap.Total)
}

// MaybeFooter returns the hint line when cap.Truncated is true, else
// "". Indented with two spaces so it sits under the list without
// looking like another row.
func MaybeFooter(cap Cap) string {
	if !cap.Truncated || cap.Hint == "" {
		return ""
	}
	return "  " + cap.Hint + "\n"
}

func exportMark(s model.SymbolWithFile) string {
	if s.Exported {
		return " exported"
	}
	return ""
}

func symLine(s model.SymbolWithFile) string {
	sig := s.Signature
	if sig != "" {
		sig = "  " + sig
	}
	return fmt.Sprintf("  %4d: %-30s  [%-6s]%s%s",
		s.StartLine, s.QualifiedName, s.Kind, exportMark(s), sig)
}

// FileOutline: list of a file's symbols ordered by line. When cap
// is truncated, items past Shown are dropped from the output and a
// header/footer is printed. The order of the input slice is
// preserved (caller is expected to sort).
func FileOutline(syms []model.SymbolWithFile, cap Cap) string {
	if len(syms) == 0 {
		return "(no symbols)"
	}
	// Truncate before formatting so the per-file grouping sees a
	// already-sized slice; this keeps the byPath map small.
	items := syms
	if cap.Truncated && cap.Shown < len(items) {
		items = items[:cap.Shown]
	}
	byPath := map[string][]model.SymbolWithFile{}
	order := []string{}
	for _, s := range items {
		if _, ok := byPath[s.FilePath]; !ok {
			order = append(order, s.FilePath)
		}
		byPath[s.FilePath] = append(byPath[s.FilePath], s)
	}
	sort.Strings(order)
	var b strings.Builder
	b.WriteString(MaybeHeader(KindSymbols, cap))
	for _, p := range order {
		fmt.Fprintf(&b, "%s\n", p)
		for _, s := range byPath[p] {
			b.WriteString(symLine(s))
			b.WriteString("\n")
		}
	}
	b.WriteString(MaybeFooter(cap))
	return b.String()
}

// PackageOutline: same shape as FileOutline, with a package header.
func PackageOutline(importPath string, syms []model.SymbolWithFile, cap Cap) string {
	items := syms
	if cap.Truncated && cap.Shown < len(items) {
		items = items[:cap.Shown]
	}
	var b strings.Builder
	b.WriteString(MaybeHeader(KindSymbols, cap))
	fmt.Fprintf(&b, "package %s  (%d symbols)\n", importPath, len(items))
	b.WriteString(FileOutline(items, formatZero(cap, len(items))))
	b.WriteString(MaybeFooter(cap))
	return b.String()
}

// formatZero returns a Cap with Truncated=false and Total/Shown set
// to n. Used internally to recurse into FileOutline without
// double-counting the header.
func formatZero(in Cap, n int) Cap {
	return Cap{Total: n, Shown: n, Truncated: false, Hint: in.Hint}
}

// RefsTo: formats incoming references (callers / uses). When cap is
// truncated, only the first Shown refs are printed.
func RefsTo(target string, refs []model.RefSite, cap Cap) string {
	items := refs
	if cap.Truncated && cap.Shown < len(items) {
		items = items[:cap.Shown]
	}
	var b strings.Builder
	if len(items) == 0 {
		return fmt.Sprintf("no references to %q", target)
	}
	b.WriteString(MaybeHeader(KindRefs, cap))
	fmt.Fprintf(&b, "references to %q  (%d sites)\n", target, len(items))
	for _, r := range items {
		fmt.Fprintf(&b, "  %s  %s:%d  [%s]\n",
			r.FromSymbol.QualifiedName, r.FromSymbol.FilePath, r.Line, r.Kind)
	}
	b.WriteString(MaybeFooter(cap))
	return b.String()
}

// RefsFrom: formats outgoing references (callees). When cap is
// truncated, only the first Shown qnames are printed.
func RefsFrom(source string, qnames []string, cap Cap) string {
	items := qnames
	if cap.Truncated && cap.Shown < len(items) {
		items = items[:cap.Shown]
	}
	var b strings.Builder
	if len(items) == 0 {
		return fmt.Sprintf("%q has no outgoing references", source)
	}
	b.WriteString(MaybeHeader(KindOutgoing, cap))
	fmt.Fprintf(&b, "outgoing references from %q  (%d)\n", source, len(items))
	for _, q := range items {
		fmt.Fprintf(&b, "  %s\n", q)
	}
	b.WriteString(MaybeFooter(cap))
	return b.String()
}

// ModuleOverview: compact table per module and package.
func ModuleOverview(mods []model.ModuleSummary, cap Cap) string {
	if len(mods) == 0 {
		return "(no modules)"
	}
	items := mods
	if cap.Truncated && cap.Shown < len(items) {
		items = items[:cap.Shown]
	}
	var b strings.Builder
	b.WriteString(MaybeHeader(KindModules, cap))
	b.WriteString("module overview\n")
	for _, m := range items {
		fmt.Fprintf(&b, "\n%s", m.ModuleID)
		if m.Dir != "" {
			fmt.Fprintf(&b, "  (dir=%s)", m.Dir)
		}
		b.WriteString("\n")
		if len(m.Packages) == 0 {
			b.WriteString("  (no packages)\n")
			continue
		}
		for _, p := range m.Packages {
			fmt.Fprintf(&b, "  %-50s  files=%-3d  syms=%-4d  exported=%d\n",
				p.PackageID, p.Files, p.Symbols, p.Exported)
		}
	}
	b.WriteString(MaybeFooter(cap))
	return b.String()
}

// Symbol: formats the definition of a symbol.
func Symbol(syms []model.SymbolWithFile) string {
	var b strings.Builder
	for _, s := range syms {
		fmt.Fprintf(&b, "%s  [%s]%s\n", s.QualifiedName, s.Kind, exportMark(s))
		fmt.Fprintf(&b, "  %s:%d-%d\n", s.FilePath, s.StartLine, s.EndLine)
		if s.Signature != "" {
			fmt.Fprintf(&b, "  signature: %s\n", s.Signature)
		}
	}
	return b.String()
}

// SymbolBody: header + numbered source lines.
func SymbolBody(sym model.SymbolWithFile, lines []model.SourceLine, maxLines int) string {
	exp := ""
	if sym.Exported {
		exp = " exported"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%d-%d  [%s]%s\n", sym.FilePath, sym.StartLine, sym.EndLine, sym.Kind, exp)
	if sym.Signature != "" {
		fmt.Fprintf(&b, "  signature: %s\n", sym.Signature)
	}
	maxLine := sym.EndLine
	if maxLines > 0 && sym.EndLine-sym.StartLine+1 > maxLines {
		maxLine = sym.StartLine + maxLines - 1
	}
	for _, l := range lines {
		fmt.Fprintf(&b, "  %4d: %s\n", l.Line, l.Content)
	}
	if maxLine < sym.EndLine {
		fmt.Fprintf(&b, "  ... truncated at line %d (max_lines=%d); symbol ends at line %d\n",
			maxLine, maxLines, sym.EndLine)
	}
	return b.String()
}

// FileRange: numbered lines with path:start-end header. No signature is
// included because the range is arbitrary (it may cross symbols).
func FileRange(path string, startLine, endLine int, lines []model.SourceLine, maxLines int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%d-%d\n", path, startLine, endLine)
	maxLine := endLine
	if maxLines > 0 && len(lines) > maxLines {
		maxLine = startLine + maxLines - 1
	}
	for _, l := range lines {
		fmt.Fprintf(&b, "  %4d: %s\n", l.Line, l.Content)
	}
	if maxLine < endLine {
		fmt.Fprintf(&b, "  ... truncated at line %d (max_lines=%d); range ends at line %d\n",
			maxLine, maxLines, endLine)
	}
	return b.String()
}
