// Package testutil holds helpers shared by tests across packages.
// Importing from a non-_test file is intentional: black-box tests
// (mekami-cli/tests/...) import it the same way production code does.
package testutil

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// ShortSockDir returns a directory suitable for binding a Unix
// domain socket. On Linux/Windows it is just t.TempDir(); on
// macOS it parks the directory under /tmp with a short name so
// the resulting socket path stays under the 104-byte sun_path
// limit and bind() does not return "invalid argument".
//
// On macOS the runtime temp dir lives under
// /var/folders/.../T/<name><digits>/<digits>/, and once you
// append .mekami/watcher.sock (or supervisor.sock) the full
// path exceeds 104 bytes. The helper works around that by
// using os.MkdirTemp under /tmp with a name truncated to 16
// chars so the final path stays well under the limit.
func ShortSockDir(t *testing.T) string {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return t.TempDir()
	}
	name := strings.ReplaceAll(t.Name(), "/", "_")
	if len(name) > 16 {
		name = name[:16]
	}
	dir, err := os.MkdirTemp("/tmp", "ms-"+name+"-")
	if err != nil {
		t.Fatalf("ShortSockDir MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}
