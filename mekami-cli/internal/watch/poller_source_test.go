package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// readEvents drains up to n events from src with a timeout. It
// returns whatever it managed to read before the deadline.
func readEvents(src Source, n int, timeout time.Duration) []Event {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	out := make([]Event, 0, n)
	for len(out) < n {
		select {
		case e, ok := <-src.Events():
			if !ok {
				return out
			}
			out = append(out, e)
		case <-deadline.C:
			return out
		}
	}
	return out
}

func TestPoller_DetectsCreate(t *testing.T) {
	dir := t.TempDir()
	src := NewPollerSource(dir, 30*time.Millisecond, StdLogger{W: nil})
	defer src.Stop()

	// Wait for the initial snapshot to settle (one tick).
	time.Sleep(60 * time.Millisecond)

	// Now create a file: the next tick must emit a Create event.
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readEvents(src, 1, 500*time.Millisecond)
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d (%v)", len(got), got)
	}
	if got[0].Path != "a.go" || got[0].Kind != EventCreate {
		t.Fatalf("unexpected event: %+v", got[0])
	}
}

func TestPoller_DetectsModify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.go")
	if err := os.WriteFile(path, []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := NewPollerSource(dir, 30*time.Millisecond, StdLogger{W: nil})
	defer src.Stop()

	// Wait for the initial snapshot to settle.
	time.Sleep(60 * time.Millisecond)

	// Modify the file. Bump mtime by at least a second because
	// some filesystems have a 1s mtime granularity.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package x\n// changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readEvents(src, 1, 500*time.Millisecond)
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d (%v)", len(got), got)
	}
	if got[0].Path != "a.go" || got[0].Kind != EventWrite {
		t.Fatalf("unexpected event: %+v", got[0])
	}
}

func TestPoller_DetectsRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.go")
	if err := os.WriteFile(path, []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	src := NewPollerSource(dir, 30*time.Millisecond, StdLogger{W: nil})
	defer src.Stop()

	time.Sleep(60 * time.Millisecond)

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	got := readEvents(src, 1, 500*time.Millisecond)
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d (%v)", len(got), got)
	}
	if got[0].Path != "a.go" || got[0].Kind != EventRemove {
		t.Fatalf("unexpected event: %+v", got[0])
	}
}

func TestPoller_SkipsIgnoredDirs(t *testing.T) {
	dir := t.TempDir()
	src := NewPollerSource(dir, 30*time.Millisecond, StdLogger{W: nil})
	defer src.Stop()

	// Wait for the initial snapshot to settle.
	time.Sleep(60 * time.Millisecond)

	// Files in hidden/build dirs must not produce events.
	for _, sub := range []string{".git", ".mekami", "node_modules", "vendor", "_dev"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, "x.go"), []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Add a real file too: only the real file should appear.
	if err := os.WriteFile(filepath.Join(dir, "real.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readEvents(src, 1, 500*time.Millisecond)
	for _, e := range got {
		for _, sub := range []string{".git", ".mekami", "node_modules", "vendor", "_dev"} {
			if len(e.Path) >= len(sub) && e.Path[:len(sub)] == sub {
				t.Errorf("poller emitted event under %q: %+v", sub, e)
			}
		}
	}
	foundReal := false
	for _, e := range got {
		if e.Path == "real.go" {
			foundReal = true
		}
	}
	if !foundReal {
		t.Errorf("expected event for real.go, got %v", got)
	}
}

func TestPoller_StopUnblocksReader(t *testing.T) {
	dir := t.TempDir()
	src := NewPollerSource(dir, 10*time.Millisecond, StdLogger{W: nil})
	if err := src.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case _, ok := <-src.Events():
		if ok {
			t.Fatalf("expected channel to be closed after Stop")
		}
	case <-time.After(time.Second):
		t.Fatalf("Events channel not closed after Stop")
	}
}
