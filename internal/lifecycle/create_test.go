package lifecycle_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dwrth/knaller/internal/allocate"
	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/lifecycle"
	"github.com/dwrth/knaller/internal/state"
)

func stubCreateOK(t *testing.T, m *lifecycle.Manager) *[]string {
	t.Helper()
	steps := stubDestroyOK(t, m)
	m.SetEnsureAcctForTest(func(name string, uid, gid int) error {
		*steps = append(*steps, fmt.Sprintf("ensureAcct:%s:%d:%d", name, uid, gid))
		return nil
	})
	m.SetProvisionStorageForTest(func(sb state.Sandbox) error {
		*steps = append(*steps, "storage:"+sb.ID)
		return nil
	})
	m.SetEnsureHostForTest(func(*config.Config) error {
		*steps = append(*steps, "ensureHost")
		return nil
	})
	m.SetStartUnitForTest(func(unit string) error {
		*steps = append(*steps, "startUnit:"+unit)
		return nil
	})
	m.SetWaitReadyForTest(func(*config.Config, state.Sandbox) error {
		*steps = append(*steps, "waitReady")
		return nil
	})
	return steps
}

func TestCreateMarksRunning(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	steps := stubCreateOK(t, m)

	sb, err := m.Create("worker-1", 2, 512)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if sb.DesiredState != state.DesiredRunning || sb.ObservedState != state.ObservedRunning {
		t.Fatalf("states = %s/%s", sb.DesiredState, sb.ObservedState)
	}

	got, err := state.New(cfg.State.Directory).Load(sb.ID)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if got.ObservedState != state.ObservedRunning {
		t.Fatalf("persisted observed = %s", got.ObservedState)
	}

	acct := allocate.AccountName(sb.Slot)
	for _, w := range []string{
		"ensureAcct:" + acct + ":12001:12001",
		"storage:" + sb.ID,
		"ensureHost",
		"startUnit:knaller@" + sb.ID + ".service",
		"waitReady",
	} {
		found := false
		for _, s := range *steps {
			if s == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("steps %v missing %s", *steps, w)
		}
	}
}

func TestCreateRollsBackOnProvisionFailure(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	stubCreateOK(t, m)
	m.SetProvisionStorageForTest(func(state.Sandbox) error {
		return fmt.Errorf("disk full")
	})

	_, err := m.Create("worker-1", 1, 256)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("Create() = %v, want disk full", err)
	}
	assertNoSandboxState(t, cfg.State.Directory)

	// Successful rollback frees the slot for reuse.
	stubCreateOK(t, m)
	sb, err := m.Create("worker-1", 1, 256)
	if err != nil {
		t.Fatalf("Create after rollback = %v", err)
	}
	if sb.Slot != 1 {
		t.Fatalf("slot after rollback = %d, want 1", sb.Slot)
	}
}

func TestCreateRollsBackOnStartFailure(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	stubCreateOK(t, m)
	m.SetStartUnitForTest(func(string) error {
		return fmt.Errorf("unit failed")
	})

	_, err := m.Create("worker-1", 1, 256)
	if err == nil || !strings.Contains(err.Error(), "unit failed") {
		t.Fatalf("Create() = %v, want unit failed", err)
	}
	assertNoSandboxState(t, cfg.State.Directory)
}

func assertNoSandboxState(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == "knaller.lock" {
			continue
		}
		if filepath.Ext(e.Name()) == ".json" {
			t.Fatalf("leftover state %s", e.Name())
		}
	}
}
