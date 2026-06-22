package naming

import "testing"

// TestShow_BodyIsNotCLIOnly locks in the contract that the
// get_symbol MCP tool advertises the `body` flag. The handler
// already reads it (see internal/handlers/read.go GetSymbol), but
// a future Spec edit that flips body back to CLIOnly would
// silently hide the flag from MCP clients. This test catches
// that regression at unit-test time.
func TestShow_BodyIsNotCLIOnly(t *testing.T) {
	spec := LookupByDispatcherKey("show")
	if spec == nil {
		t.Fatal("show spec not found")
	}
	for _, f := range spec.Flags {
		if f.Name != "body" {
			continue
		}
		if f.CLIOnly {
			t.Errorf("show.body must NOT be CLIOnly; the get_symbol MCP tool must advertise it")
		}
		return
	}
	t.Errorf("show spec is missing the `body` flag entirely")
}

// TestShow_JSONRemainsCLIOnly is the inverse: --json is a CLI
// output-format switch, not an MCP argument. The MCP server has
// its own json-handling path; exposing `json` on the schema would
// invite the LLM to pass it and get a JSON blob back when it
// asked for text.
func TestShow_JSONRemainsCLIOnly(t *testing.T) {
	spec := LookupByDispatcherKey("show")
	if spec == nil {
		t.Fatal("show spec not found")
	}
	for _, f := range spec.Flags {
		if f.Name != "json" {
			continue
		}
		if !f.CLIOnly {
			t.Errorf("show.json MUST remain CLIOnly; do not advertise it on the MCP schema")
		}
		return
	}
	t.Errorf("show spec is missing the `json` flag entirely")
}
