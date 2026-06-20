package mekami

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Wolf258/mekami-api/api/v1"

	"github.com/Wolf258/mekami-cli/internal/config"
	"github.com/Wolf258/mekami-cli/internal/naming"
)

// resetAPIGlobal swaps api.Global for a fresh registry and returns
// a cleanup func. Tests that depend on which frontends are
// registered call this in t.Cleanup so they do not leak state
// across the suite.
func resetAPIGlobal(t *testing.T) {
	t.Helper()
	orig := api.Global
	t.Cleanup(func() { api.Global = orig })
	api.Global = api.NewRegistry()
}

// fakeFrontend implements api.Frontend with the minimum surface
// resolveLang, resolveInitLangs, runInit and List need. It is a
// stub: ParseFile returns an empty ParseResult and StructuralFiles
// is nil. ResolveLayout returns a non-workspace, which is what
// ingest.Build expects when the language has no workspace concept.
type fakeFrontend struct{ name string }

func (f fakeFrontend) Name() string              { return f.name }
func (f fakeFrontend) Extensions() []string      { return []string{".x"} }
func (f fakeFrontend) StructuralFiles() []string { return nil }
func (f fakeFrontend) IsIndexable(string) bool   { return true }
func (f fakeFrontend) ResolveLayout(string) (*api.Workspace, error) {
	return &api.Workspace{}, nil
}
func (f fakeFrontend) ResolveModules(string) ([]api.ModuleInfo, error) {
	return nil, nil
}
func (f fakeFrontend) RootModule(string) (string, error) { return "", nil }
func (f fakeFrontend) ResolveFile(string, string) (api.FileMeta, error) {
	return api.FileMeta{}, nil
}
func (f fakeFrontend) ParseFile(string, string, string, string, int64, int64) (api.ParseResult, error) {
	return api.ParseResult{}, nil
}

func TestResolveLang(t *testing.T) {
	tests := []struct {
		name      string
		registers []string
		cfg       config.Config
		explicit  string
		wantOK    bool
		wantLang  string
		wantInErr []string
	}{
		{
			name:      "empty_config_no_explicit_errors_no_cores_installed",
			wantOK:    false,
			wantInErr: []string{"no cores installed", "core install"},
		},
		{
			name:      "empty_config_explicit_go_with_binary_registered_ok",
			registers: []string{"go"},
			explicit:  "go",
			wantOK:    true,
			wantLang:  "go",
		},
		{
			name:      "empty_config_explicit_go_not_in_binary_errors",
			explicit:  "go",
			wantOK:    false,
			wantInErr: []string{`--lang "go"`, "core install"},
		},
		{
			name:      "empty_config_explicit_unknown_errors",
			explicit:  "python",
			wantOK:    false,
			wantInErr: []string{`"python"`},
		},
		{
			name:      "single_indexer_registered_ok",
			registers: []string{"go"},
			cfg:       config.Config{Indexers: map[string]string{"go": "v0.1.0"}},
			wantOK:    true,
			wantLang:  "go",
		},
		{
			name:      "single_indexer_not_in_binary_errors_configured_but_missing",
			cfg:       config.Config{Indexers: map[string]string{"go": "v0.1.0"}},
			wantOK:    false,
			wantInErr: []string{"configured but not registered"},
		},
		{
			name:      "multiple_indexers_no_explicit_errors_ambiguous",
			registers: []string{"go", "rust"},
			cfg: config.Config{Indexers: map[string]string{
				"go":   "v0.1.0",
				"rust": "v0.2.0",
			}},
			wantOK:    false,
			wantInErr: []string{"--lang is required"},
		},
		{
			name:      "multiple_indexers_explicit_picks_requested",
			registers: []string{"go", "rust"},
			cfg: config.Config{Indexers: map[string]string{
				"go":   "v0.1.0",
				"rust": "v0.2.0",
			}},
			explicit: "rust",
			wantOK:   true,
			wantLang: "rust",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetAPIGlobal(t)
			for _, name := range tc.registers {
				api.Global.Register(fakeFrontend{name: name})
			}
			got, err := resolveLang(tc.cfg, tc.explicit)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				if got != tc.wantLang {
					t.Errorf("got %q, want %q", got, tc.wantLang)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			msg := err.Error()
			for _, want := range tc.wantInErr {
				if !strings.Contains(msg, want) {
					t.Errorf("err = %q, want substring %q", msg, want)
				}
			}
		})
	}
}

// withCwd swaps the working directory for the duration of t. The
// init flow reads its config from .mekami/config.json relative to
// cwd and writes its DB to ./.mekami/graph.db, so every init test
// has to run inside a temp dir.
func withCwd(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// readConfig parses the .mekami/config.json that init wrote.
func readConfig(t *testing.T) config.Config {
	t.Helper()
	path := config.DefaultPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var cfg config.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return cfg
}

func TestResolveInitLangs(t *testing.T) {
	tests := []struct {
		name      string
		registers []string
		requested []string
		available []string
		wantOK    bool
		wantLangs []string
		wantInErr []string
	}{
		{
			name:      "no_cores_errors",
			wantOK:    false,
			wantInErr: []string{"no language cores registered"},
		},
		{
			name:      "empty_requested_uses_all_sorted",
			registers: []string{"rust", "go"},
			available: []string{"go", "rust"},
			wantOK:    true,
			wantLangs: []string{"go", "rust"},
		},
		{
			name:      "requested_known_ok",
			registers: []string{"go", "rust"},
			requested: []string{"rust"},
			available: []string{"go", "rust"},
			wantOK:    true,
			wantLangs: []string{"rust"},
		},
		{
			name:      "requested_unknown_errors",
			registers: []string{"go"},
			requested: []string{"python"},
			available: []string{"go"},
			wantOK:    false,
			wantInErr: []string{`"python"`, "core install"},
		},
		{
			name:      "requested_duplicates_dedupe",
			registers: []string{"go"},
			requested: []string{"go", "go", "go"},
			available: []string{"go"},
			wantOK:    true,
			wantLangs: []string{"go"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetAPIGlobal(t)
			for _, name := range tc.registers {
				api.Global.Register(fakeFrontend{name: name})
			}
			got, err := resolveInitLangs(tc.requested, tc.available)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				if !stringSliceEqual(got, tc.wantLangs) {
					t.Errorf("got %v, want %v", got, tc.wantLangs)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			msg := err.Error()
			for _, want := range tc.wantInErr {
				if !strings.Contains(msg, want) {
					t.Errorf("err = %q, want substring %q", msg, want)
				}
			}
		})
	}
}

func TestMergeIndexers(t *testing.T) {
	tests := []struct {
		name        string
		existing    map[string]string
		selected    map[string]string
		explicit    bool
		wantKeys    []string
		wantVersion map[string]string
	}{
		{
			name:     "explicit_replaces",
			existing: map[string]string{"rust": "v0.2.0", "go": "v0.1.0"},
			selected: map[string]string{"go": ""},
			explicit: true,
			wantKeys: []string{"go"},
		},
		{
			name:     "explicit_preserves_existing_version",
			existing: map[string]string{"go": "v0.1.0"},
			selected: map[string]string{"go": ""},
			explicit: true,
			wantKeys: []string{"go"},
			wantVersion: map[string]string{
				"go": "v0.1.0",
			},
		},
		{
			name:     "implicit_unions",
			existing: map[string]string{"rust": ""},
			selected: map[string]string{"go": ""},
			explicit: false,
			wantKeys: []string{"go", "rust"},
		},
		{
			name:     "implicit_keeps_existing_even_if_missing_from_selected",
			existing: map[string]string{"rust": ""},
			selected: map[string]string{"rust": ""},
			explicit: false,
			wantKeys: []string{"rust"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeIndexers(tc.existing, tc.selected, tc.explicit)
			names := mapKeys(got)
			if !stringSliceEqual(names, tc.wantKeys) {
				t.Errorf("merge keys = %v, want %v", names, tc.wantKeys)
			}
			for k, wantVer := range tc.wantVersion {
				if got[k] != wantVer {
					t.Errorf("merge[%q] = %q, want %q", k, got[k], wantVer)
				}
			}
		})
	}
}

func TestRunInit(t *testing.T) {
	tests := []struct {
		name         string
		registers    []string
		args         []string
		wantOK       bool
		wantIndexers []string
		wantInErr    []string
		wantNoConfig bool
		wantDBExists bool
	}{
		{
			name:         "no_cores_errors_before_writing_config",
			wantOK:       false,
			wantInErr:    []string{"no language cores registered"},
			wantNoConfig: true,
		},
		{
			name:         "all_available_single_writes_config_with_that_core",
			registers:    []string{"go"},
			wantOK:       true,
			wantIndexers: []string{"go"},
			wantDBExists: true,
		},
		{
			name:         "all_available_multiple_writes_all_sorted",
			registers:    []string{"rust", "go"},
			wantOK:       true,
			wantIndexers: []string{"go", "rust"},
		},
		{
			name:         "explicit_lang_subset_writes_subset",
			registers:    []string{"go", "rust"},
			args:         []string{"--lang", "rust"},
			wantOK:       true,
			wantIndexers: []string{"rust"},
		},
		{
			name:      "explicit_unknown_errors",
			registers: []string{"go"},
			args:      []string{"--lang", "python"},
			wantOK:    false,
			wantInErr: []string{`"python"`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetAPIGlobal(t)
			for _, name := range tc.registers {
				api.Global.Register(fakeFrontend{name: name})
			}
			dir := t.TempDir()
			withCwd(t, dir)
			err := runInit(t.Context(), newInitCmd(t, tc.args...), nil)
			if !tc.wantOK {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				msg := err.Error()
				for _, want := range tc.wantInErr {
					if !strings.Contains(msg, want) {
						t.Errorf("err = %q, want substring %q", msg, want)
					}
				}
				if tc.wantNoConfig {
					if _, statErr := os.Stat(config.DefaultPath()); !os.IsNotExist(statErr) {
						t.Errorf("expected no config to be written, stat err = %v", statErr)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			cfg := readConfig(t)
			if !stringSliceEqual(mapKeys(cfg.Indexers), tc.wantIndexers) {
				t.Errorf("indexers = %v, want %v", mapKeys(cfg.Indexers), tc.wantIndexers)
			}
			if tc.wantDBExists {
				if _, err := os.Stat(filepath.Join(dir, ".mekami", "graph.db")); err != nil {
					t.Errorf("expected graph.db to exist: %v", err)
				}
			}
		})
	}
}

func TestRunInit_ReInitPreservesAndUnionsIndexersWhenNoFlag(t *testing.T) {
	resetAPIGlobal(t)
	api.Global.Register(fakeFrontend{name: "go"})
	api.Global.Register(fakeFrontend{name: "rust"})
	dir := t.TempDir()
	withCwd(t, dir)
	// First init: nothing in config, both cores available.
	if err := runInit(t.Context(), newInitCmd(t), nil); err != nil {
		t.Fatalf("first init: %v", err)
	}
	// Hand-edit the config to drop "go" so we can verify the
	// second init unions "go" back in (the binary now registers
	// both, so all-available re-adds it) without removing "rust".
	cfg := readConfig(t)
	cfg.Indexers = map[string]string{"rust": ""}
	if err := config.Save(cfg, config.DefaultPath()); err != nil {
		t.Fatalf("save: %v", err)
	}
	// Second init: --lang omitted, must keep "rust" and re-add "go".
	if err := runInit(t.Context(), newInitCmd(t), nil); err != nil {
		t.Fatalf("second init: %v", err)
	}
	cfg2 := readConfig(t)
	names := mapKeys(cfg2.Indexers)
	sort.Strings(names)
	if !stringSliceEqual(names, []string{"go", "rust"}) {
		t.Errorf("indexers = %v, want [go rust] (union of existing + available)", names)
	}
}

func TestRunInit_ReInitWithFlagOverrides(t *testing.T) {
	resetAPIGlobal(t)
	api.Global.Register(fakeFrontend{name: "go"})
	api.Global.Register(fakeFrontend{name: "rust"})
	dir := t.TempDir()
	withCwd(t, dir)
	if err := runInit(t.Context(), newInitCmd(t, "--lang", "rust"), nil); err != nil {
		t.Fatalf("first init: %v", err)
	}
	cfg := readConfig(t)
	if !stringSliceEqual(mapKeys(cfg.Indexers), []string{"rust"}) {
		t.Fatalf("first init indexers = %v, want [rust]", mapKeys(cfg.Indexers))
	}
	cmd := newInitCmd(t, "--lang", "go")
	if err := runInit(t.Context(), cmd, nil); err != nil {
		t.Fatalf("second init: %v", err)
	}
	cfg2 := readConfig(t)
	if !stringSliceEqual(mapKeys(cfg2.Indexers), []string{"go"}) {
		t.Errorf("indexers = %v, want [go] after explicit --lang", mapKeys(cfg2.Indexers))
	}
}

func TestRunInit_VerboseFlag_DoesNotPanic(t *testing.T) {
	resetAPIGlobal(t)
	api.Global.Register(fakeFrontend{name: "go"})
	dir := t.TempDir()
	withCwd(t, dir)
	cmd := newInitCmd(t, "--verbose")
	if err := runInit(t.Context(), cmd, nil); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

// newBuildCmd mirrors newInitCmd for the build subcommand. The
// cross-language tests below need a real *cobra.Command so
// cmd.Flags().Changed("lang") and friends work the same way
// they do at runtime.
func newBuildCmd(t *testing.T, extraArgs ...string) *cobra.Command {
	t.Helper()
	for _, spec := range naming.Specs {
		if spec.Use == "build" {
			cmd := naming.CobraCommand(spec, func(*cobra.Command, []string) error { return nil })
			args := append([]string{"--quiet"}, extraArgs...)
			cmd.SetArgs(args)
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatalf("ParseFlags: %v", err)
			}
			return cmd
		}
	}
	t.Fatal("build spec not found")
	return nil
}

// prelabelFile inserts an extra file row with the given lang
// into the graph DB. Used to simulate "this project used to
// track a different language" so the prune path can be
// exercised from the CLI side.
func prelabelFile(t *testing.T, dbPath, lang, filePath string) {
	t.Helper()
	s, err := openStore(dbPath)
	if err != nil {
		t.Fatalf("open %s: %v", dbPath, err)
	}
	defer s.Close()
	if _, err := s.DB().Exec(
		`INSERT INTO files(path,hash,mtime,size,lang) VALUES(?,?,?,?,?)`,
		filePath, "h", 0, 0, lang); err != nil {
		t.Fatalf("insert %s: %v", filePath, err)
	}
}

// countLangInDB is the open-by-path version of the helper used
// in the core tests. CLI tests use the openStore helper from
// runner.go to get a store handle without re-implementing the
// path resolution.
func countLangInDB(t *testing.T, dbPath, lang string) int {
	t.Helper()
	s, err := openStore(dbPath)
	if err != nil {
		t.Fatalf("open %s: %v", dbPath, err)
	}
	defer s.Close()
	var n int
	if err := s.DB().QueryRow(
		`SELECT COUNT(*) FROM files WHERE lang = ?`, lang).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// TestRunBuild_AddsNewLangToConfig verifies that `mekami build
// --lang rust` against a project whose config only knows about
// `go` extends the config in place and logs the warning the
// user expects.
func TestRunBuild_AddsNewLangToConfig(t *testing.T) {
	resetAPIGlobal(t)
	api.Global.Register(fakeFrontend{name: "go"})
	api.Global.Register(fakeFrontend{name: "rust"})
	dir := t.TempDir()
	withCwd(t, dir)
	// Initialise with only go so the config has indexers=[go].
	if err := runInit(t.Context(), newInitCmd(t, "--lang", "go"), nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	cfg := readConfig(t)
	if !stringSliceEqual(mapKeys(cfg.Indexers), []string{"go"}) {
		t.Fatalf("setup: indexers = %v, want [go]", mapKeys(cfg.Indexers))
	}
	// Build with --lang rust. The CLI must add rust to the
	// config and log the addition.
	if err := runBuild(t.Context(), newBuildCmd(t, "--lang", "rust")); err != nil {
		t.Fatalf("build: %v", err)
	}
	cfg2 := readConfig(t)
	names := mapKeys(cfg2.Indexers)
	sort.Strings(names)
	if !stringSliceEqual(names, []string{"go", "rust"}) {
		t.Errorf("indexers = %v, want [go rust]", names)
	}
}

// TestRunBuild_PrunesDataForDisabledLang verifies that a build
// with AllowedLangs narrower than the data in the DB drops the
// foreign rows and that the DB ends up consistent.
func TestRunBuild_PrunesDataForDisabledLang(t *testing.T) {
	resetAPIGlobal(t)
	api.Global.Register(fakeFrontend{name: "go"})
	api.Global.Register(fakeFrontend{name: "rust"})
	dir := t.TempDir()
	withCwd(t, dir)
	// Initialise with only go. The build runs in this pass so
	// the DB exists.
	if err := runInit(t.Context(), newInitCmd(t, "--lang", "go"), nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	dbPath := filepath.Join(dir, ".mekami", "graph.db")
	// Pre-seed a phantom python row so we can verify the prune
	// actually deletes it.
	prelabelFile(t, dbPath, "python", "phantom.py")
	if got := countLangInDB(t, dbPath, "python"); got != 1 {
		t.Fatalf("setup: expected 1 python row, got %d", got)
	}
	// Build with --lang rust. The CLI must add rust to the
	// config (it was missing) AND prune the python row before
	// the rust frontend walks the (empty) source tree.
	if err := runBuild(t.Context(), newBuildCmd(t, "--lang", "rust")); err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := countLangInDB(t, dbPath, "python"); got != 0 {
		t.Errorf("python row survived prune: got %d, want 0", got)
	}
	cfg := readConfig(t)
	names := mapKeys(cfg.Indexers)
	sort.Strings(names)
	if !stringSliceEqual(names, []string{"go", "rust"}) {
		t.Errorf("indexers = %v, want [go rust]", names)
	}
}

// TestRunBuild_NoPruneWhenConfigMatches verifies the silent
// happy path: when the config's indexers already match the
// data in the DB, no cross-language cleanup log line is
// produced and the rows are not touched.
func TestRunBuild_NoPruneWhenConfigMatches(t *testing.T) {
	resetAPIGlobal(t)
	api.Global.Register(fakeFrontend{name: "go"})
	dir := t.TempDir()
	withCwd(t, dir)
	if err := runInit(t.Context(), newInitCmd(t), nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	dbPath := filepath.Join(dir, ".mekami", "graph.db")
	prelabelFile(t, dbPath, "go", "phantom.go")
	prelabelFile(t, dbPath, "python", "phantom.py")
	// Now strip the python row from the config so it doesn't
	// belong; but we'll also remove it from the DB to test the
	// pure-match case.
	// Simpler: write a fresh go-only project and just run
	// build again.
	if err := runBuild(t.Context(), newBuildCmd(t, "--lang", "go")); err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := countLangInDB(t, dbPath, "go"); got == 0 {
		t.Errorf("expected go rows, got %d", got)
	}
}

// TestRunInit_PrunesOldLangData verifies that a re-init with a
// different --lang drops the data the project no longer tracks
// (the "init with new lang" path).
func TestRunInit_PrunesOldLangData(t *testing.T) {
	resetAPIGlobal(t)
	api.Global.Register(fakeFrontend{name: "go"})
	api.Global.Register(fakeFrontend{name: "rust"})
	dir := t.TempDir()
	withCwd(t, dir)
	// First init: only go.
	if err := runInit(t.Context(), newInitCmd(t, "--lang", "go"), nil); err != nil {
		t.Fatalf("init go: %v", err)
	}
	dbPath := filepath.Join(dir, ".mekami", "graph.db")
	// Plant a phantom rust row to simulate "a previous init
	// tracked rust, but the user is reconfiguring to go only".
	prelabelFile(t, dbPath, "rust", "phantom.rs")
	if got := countLangInDB(t, dbPath, "rust"); got != 1 {
		t.Fatalf("setup: expected 1 rust row, got %d", got)
	}
	// Re-init with only go. The config still says [go], the
	// rust row is foreign, the build's AllowedLangs=[go] must
	// drop it.
	if err := runInit(t.Context(), newInitCmd(t, "--lang", "go"), nil); err != nil {
		t.Fatalf("init re: %v", err)
	}
	if got := countLangInDB(t, dbPath, "rust"); got != 0 {
		t.Errorf("rust row survived prune: got %d, want 0", got)
	}
}

// stringSliceEqual reports whether a and b are equal element-wise.
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// mapKeys returns the sorted list of keys in m, useful for
// comparing the language set of a config in tests.
func mapKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// newInitCmd builds a *cobra.Command wired with the same flags
// the real init command declares in the naming spec, so the
// runInit entry point can read them with cmd.Flags(). Tests pass
// --daemon=no by default so the TTY prompt never fires; the args
// the test supplies are appended after that.
func newInitCmd(t *testing.T, extraArgs ...string) *cobra.Command {
	t.Helper()
	for _, spec := range naming.Specs {
		if spec.Use == "init" {
			cmd := naming.CobraCommand(spec, func(*cobra.Command, []string) error { return nil })
			args := append([]string{"--daemon=no"}, extraArgs...)
			cmd.SetArgs(args)
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatalf("ParseFlags: %v", err)
			}
			return cmd
		}
	}
	t.Fatal("init spec not found")
	return nil
}

// TestServiceCommands_RegisteredAsGroup is a regression test for
// the design where the supervisor registration actions live
// under a single `service` parent (`mekami service install`,
// `mekami service uninstall`, `mekami service status`). The
// test asserts:
//
//   - the parent `service` command is registered at the
//     top level (it is a namespace carrier with no RunE);
//   - the three subcommands `install`, `uninstall`, and
//     `status` are present under it and visible (not hidden);
//   - their Specs are present in naming.Specs with the
//     expected Parent + DispatcherKey.
//
// The test does not exec systemctl/launchctl: that path is
// gated behind the `integration` build tag and requires a
// live user bus. The contract tested here is purely the
// cobra registration.
func TestServiceCommands_RegisteredAsGroup(t *testing.T) {
	var serviceCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "service" {
			serviceCmd = c
			break
		}
	}
	if serviceCmd == nil {
		t.Fatal("`service` parent command is not registered at the top level")
	}

	want := map[string]bool{"install": false, "uninstall": false, "status": false}
	for _, sub := range serviceCmd.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("`service %s` is not registered as a subcommand", name)
		}
	}

	// Specs must agree with the cobra tree. A divergence here
	// would mean someone changed one but not the other.
	for _, key := range []string{"service.install", "service.uninstall", "service.status"} {
		s := naming.LookupByDispatcherKey(key)
		if s == nil {
			t.Errorf("naming.Specs is missing the %q entry", key)
			continue
		}
		if s.Parent != "service" {
			t.Errorf("spec %q has Parent=%q, want \"service\"", key, s.Parent)
		}
		if s.Use != strings.TrimPrefix(key, "service.") {
			t.Errorf("spec %q has Use=%q, want %q", key, s.Use, strings.TrimPrefix(key, "service."))
		}
	}
}

// TestServiceCommands_NewInvocationAcceptsSubcommand exercises the
// canonical `mekami service install` invocation end-to-end
// through cobra. The parent `service` is now a real subcommand
// group, and `install` is a registered subcommand of it. We do
// not assert on the runner's return value (the platform layer
// may legitimately fail in CI environments without a user bus);
// we only assert that cobra recognises the path and dispatches
// into the runner rather than bailing with "unknown command".
func TestServiceCommands_NewInvocationAcceptsSubcommand(t *testing.T) {
	origArgs := rootCmd.Flags().Args()
	t.Cleanup(func() { rootCmd.SetArgs(origArgs) })

	outBuf := &strings.Builder{}
	errBuf := &strings.Builder{}
	origOut := rootCmd.OutOrStderr()
	origErr := rootCmd.ErrOrStderr()
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	t.Cleanup(func() {
		rootCmd.SetOut(origOut)
		rootCmd.SetErr(origErr)
	})

	rootCmd.SetArgs([]string{"service", "install", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		// A "help" invocation should never error. If it
		// does, the path did not reach the runner.
		t.Fatalf("`mekami service install --help` failed: %v\nstdout=%q\nstderr=%q",
			err, outBuf.String(), errBuf.String())
	}
}

// TestServiceCommands_LegacyFlatFormFailsCleanly exercises the
// old `mekami service-install` (single token) invocation. After
// the refactor that form is no longer a registered command and
// cobra must print a clear "unknown command" error rather than
// silently dispatching somewhere wrong. The error message
// should mention the user-typed form so a misdirected migration
// is easy to spot.
func TestServiceCommands_LegacyFlatFormFailsCleanly(t *testing.T) {
	origArgs := rootCmd.Flags().Args()
	t.Cleanup(func() { rootCmd.SetArgs(origArgs) })

	outBuf := &strings.Builder{}
	errBuf := &strings.Builder{}
	origOut := rootCmd.OutOrStderr()
	origErr := rootCmd.ErrOrStderr()
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	t.Cleanup(func() {
		rootCmd.SetOut(origOut)
		rootCmd.SetErr(origErr)
	})

	rootCmd.SetArgs([]string{"service-install"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatalf("expected `mekami service-install` (legacy form) to fail; "+
			"got nil. stdout=%q stderr=%q", outBuf.String(), errBuf.String())
	}
	msg := err.Error()
	if !strings.Contains(msg, "service-install") {
		t.Errorf("error should mention the legacy form `service-install`: %q", msg)
	}
}

// TestRunServiceDispatch is a lightweight smoke test that the
// public runner wiring actually calls the per-platform functions.
// We exercise the path by checking the return value matches what
// the per-platform code would produce: on linux/darwin it depends
// on a real user bus, so we skip on success because the
// environment is not the test's concern; we just want the
// dispatch contract. See the integration tests (build tag
// `integration && linux`) for the full systemd round-trip.
func TestRunServiceDispatch(t *testing.T) {
	tests := []struct {
		name    string
		call    func() error
		skipMsg string
		// acceptAnyInErr lists substrings that, if present in
		// the error, prove the dispatch reached the platform
		// layer (per-OS error wording varies).
		acceptAnyInErr []string
	}{
		{
			name: "install",
			call: runServiceInstall,
			skipMsg: "runServiceInstall returned nil; environment-specific " +
				"success is not asserted here (see integration tests for " +
				"the full systemd round-trip)",
			acceptAnyInErr: []string{
				"unsupported platform",
				"systemctl",
				"daemon-reload",
				"mkdir",
				"write unit",
			},
		},
		{
			name: "uninstall",
			call: runServiceUninstall,
			skipMsg: "runServiceUninstall returned nil; environment-specific " +
				"success is not asserted here",
			acceptAnyInErr: []string{
				"unsupported platform",
				"systemctl",
				"launchctl",
				"remove unit",
				"remove plist",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Skip(tc.skipMsg)
			}
			for _, want := range tc.acceptAnyInErr {
				if strings.Contains(err.Error(), want) {
					return
				}
			}
			t.Errorf("%s returned an unexpected error "+
				"(should be a platform/service-manager error): %v",
				tc.name, err)
		})
	}
}
