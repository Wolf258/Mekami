package supervisor

import (
	"sync"
	"testing"
)

func TestInotifyBudget_UsageAccounting(t *testing.T) {
	cases := []struct {
		name        string
		ops         [][2]any // [root, n] tuples to apply
		wantUsage   int64
		wantMissing string
	}{
		{
			name:      "set_and_total",
			ops:       [][2]any{{"/proj/a", int64(100)}, {"/proj/b", int64(250)}},
			wantUsage: 350,
		},
		{
			name:      "replace_value",
			ops:       [][2]any{{"/proj/a", int64(100)}, {"/proj/a", int64(50)}, {"/proj/a", int64(75)}},
			wantUsage: 75,
		},
		{
			name:        "remove_on_zero",
			ops:         [][2]any{{"/proj/a", int64(100)}, {"/proj/b", int64(50)}, {"/proj/a", int64(0)}},
			wantUsage:   50,
			wantMissing: "/proj/a",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := NewInotifyBudget()
			for _, op := range c.ops {
				root, _ := op[0].(string)
				n, _ := op[1].(int64)
				b.SetDaemonWatches(root, n)
			}
			if got := b.Usage(); got != c.wantUsage {
				t.Fatalf("usage = %d, want %d", got, c.wantUsage)
			}
			if c.wantMissing != "" {
				if _, ok := b.perDaemon[c.wantMissing]; ok {
					t.Fatalf("expected %s to be removed from perDaemon", c.wantMissing)
				}
			}
		})
	}
}

func TestInotifyBudget_LevelBuckets(t *testing.T) {
	cases := []struct {
		name  string
		usage int64
		want  BudgetLevel
	}{
		{"ok_low", 0, BudgetOK},
		{"ok_below_warning", 599, BudgetOK},
		{"warning_low", 600, BudgetWarning},
		{"warning_below_degraded", 799, BudgetWarning},
		{"degraded_low", 800, BudgetDegraded},
		{"degraded_below_critical", 949, BudgetDegraded},
		{"critical_low", 950, BudgetCritical},
		{"critical_at_limit", 1000, BudgetCritical},
		{"critical_above_limit", 2000, BudgetCritical},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := &InotifyBudget{
				limit:     1000,
				perDaemon: make(map[string]int64),
			}
			b.usage = c.usage
			if got := b.Level(); got != c.want {
				t.Errorf("usage=%d: got %v, want %v", c.usage, got, c.want)
			}
		})
	}
}

func TestInotifyBudget_UnknownLevel(t *testing.T) {
	// Build a budget with an unknown limit directly; on Linux
	// NewInotifyBudget would probe /proc and find a real value.
	b := &InotifyBudget{limit: -1, perDaemon: make(map[string]int64)}
	b.SetDaemonWatches("/p", 100)
	if b.Level() != BudgetUnknown {
		t.Fatalf("expected BudgetUnknown with limit=-1, got %v", b.Level())
	}
	if b.Percent() != -1 {
		t.Fatalf("expected Percent=-1 with limit=-1")
	}
}

func TestInotifyBudget_SuggestPollingTargets(t *testing.T) {
	b := NewInotifyBudget()
	b.SetDaemonWatches("/small", 10)
	b.SetDaemonWatches("/big", 1000)
	b.SetDaemonWatches("/medium", 500)
	got := b.SuggestPollingTargets(2)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].Root != "/big" || got[1].Root != "/medium" {
		t.Fatalf("ordering wrong: %+v", got)
	}
}

func TestInotifyBudget_ConcurrentSafe(t *testing.T) {
	b := NewInotifyBudget()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			b.SetDaemonWatches("/p", 1)
		}(i)
		go func() {
			defer wg.Done()
			_ = b.Level()
		}()
	}
	wg.Wait()
	if got := b.Usage(); got != 1 {
		t.Fatalf("expected final usage=1 (replace semantics), got %d", got)
	}
}
