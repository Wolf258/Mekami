package supervisor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// filepathJoinTemp returns a path inside t.TempDir(). It's a
// convenience for tests that need a stable file path.
func filepathJoinTemp(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "daemons.json")
}

func filepathJoin(parts ...string) string {
	return filepath.Join(parts...)
}

// writeFile writes content to path. Thin wrapper to keep the
// tests focused on behaviour.
func writeFile(path, content string) (int, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return 0, err
	}
	return len(content), os.WriteFile(path, []byte(content), 0o644)
}

// shortSockDir returns a directory in which to put a Unix
// domain socket for the current test. On Linux and Windows
// t.TempDir() is short enough. On macOS the runtime temp dir
// lives under /var/folders/.../T/<name><digits>/<digits>/
// which, combined with our .mekami/watcher.sock (or
// supervisor.sock) suffix, easily pushes the full path past
// the 104-byte sun_path limit and bind() returns
// "invalid argument". We work around that by parking the
// socket in /tmp with a short, test-derived name. The dir is
// registered for cleanup.
func shortSockDir(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return t.TempDir()
	}
	// t.Name() can be arbitrarily long when subtests are
	// nested. Truncate to a prefix that keeps the final
	// /tmp/ms-<name>-XXXXXXXX/<sock> well under 104 bytes.
	name := strings.ReplaceAll(t.Name(), "/", "_")
	if len(name) > 16 {
		name = name[:16]
	}
	dir, err := os.MkdirTemp("/tmp", "ms-"+name+"-")
	if err != nil {
		t.Fatalf("shortSockDir MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
