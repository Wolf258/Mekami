package mcp

import (
	"testing"

	"github.com/Wolf258/mekami-cli/internal/naming"
)

// TestBuildInputSchema_GetSymbolExposesBody verifies the `body`
// flag is advertised on the get_symbol tool. This is the contract
// the LLM relies on: without it, the schema would silently hide a
// flag the handler already accepts and the test smoke (which
// passes body=true) would still work, but real clients would not
// know the flag exists.
func TestBuildInputSchema_GetSymbolExposesBody(t *testing.T) {
	spec := naming.LookupByName("get_symbol")
	if spec == nil {
		t.Fatal("get_symbol spec not found")
	}
	schema := buildInputSchema(*spec)
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema has no properties: %#v", schema)
	}
	body, ok := props["body"]
	if !ok {
		t.Fatal("get_symbol schema is missing the `body` property; " +
			"the spec must drop CLIOnly: true from the body flag so MCP clients see it")
	}
	bm, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("body is not a schema fragment: %#v", body)
	}
	if bm["type"] != "boolean" {
		t.Errorf("body type = %v, want boolean", bm["type"])
	}
}

// TestBuildInputSchema_GetSymbolHidesHeader asserts the inverse:
// --header is CLIOnly and must not appear in the MCP schema. The
// handler does not read it (it's dead code), so exposing it would
// advertise a no-op.
func TestBuildInputSchema_GetSymbolHidesHeader(t *testing.T) {
	spec := naming.LookupByName("get_symbol")
	if spec == nil {
		t.Fatal("get_symbol spec not found")
	}
	schema := buildInputSchema(*spec)
	props := schema["properties"].(map[string]any)
	if _, ok := props["header"]; ok {
		t.Errorf("`header` must remain CLIOnly; the handler does not read it")
	}
}

// TestBuildInputSchema_JSONRemainsCLIOnly locks in that --json is
// NOT advertised on the MCP schema. The MCP server already honors
// an implicit json preference via dispatch; letting the LLM set
// `json=true` would produce a JSON blob the LLM cannot read
// cleanly (it asked for text by default).
func TestBuildInputSchema_JSONRemainsCLIOnly(t *testing.T) {
	for _, name := range []string{"get_symbol", "who_calls", "list_files", "dependents"} {
		spec := naming.LookupByName(name)
		if spec == nil {
			t.Fatalf("%s spec not found", name)
		}
		schema := buildInputSchema(*spec)
		props := schema["properties"].(map[string]any)
		if _, ok := props["json"]; ok {
			t.Errorf("%s schema should not expose `json`; it is CLI-only", name)
		}
	}
}

// TestBuildInputSchema_FlagsMatchExposedSet is a regression test
// that catches accidental regressions on every graph-read tool.
// For each tool, the exposed properties must match the Spec's
// non-CLIOnly flags (plus the args).
func TestBuildInputSchema_FlagsMatchExposedSet(t *testing.T) {
	for i := range naming.Specs {
		spec := &naming.Specs[i]
		if spec.Name == "" {
			continue
		}
		schema := buildInputSchema(*spec)
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s: schema has no properties", spec.Name)
		}
		// Every exposed prop must come from either an arg or a
		// non-CLIOnly flag. If a prop is here that has no
		// source, the schema builder is leaking something.
		for name := range props {
			found := false
			for _, a := range spec.Args {
				if a.Name == name {
					found = true
					break
				}
			}
			if !found {
				for _, f := range spec.Flags {
					if f.Name == name && !f.CLIOnly {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("%s: schema property %q has no source in Args/Flags", spec.Name, name)
			}
		}
		// Required list must list exactly the args.
		req, _ := schema["required"].([]string)
		if len(req) != len(spec.Args) {
			t.Errorf("%s: required = %v, want %d args", spec.Name, req, len(spec.Args))
		}
		for i, a := range spec.Args {
			if i >= len(req) || req[i] != a.Name {
				t.Errorf("%s: required[%d] = %q, want %q", spec.Name, i, req[i], a.Name)
			}
		}
	}
}
