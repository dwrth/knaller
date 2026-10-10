package lifecycle_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dwrth/knaller/internal/allocate"
	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/lifecycle"
	"github.com/dwrth/knaller/internal/state"
)

func stubDestroyOK(t *testing.T, m *lifecycle.Manager) *[]string {
	t.Helper()
	var steps []string
	m.SetStopUnitForTest(func(unit string) error {
		steps = append(steps, "stopUnit:"+unit)
		return nil
	})
	m.SetStopRuntimeForTest(func(*config.Config, state.Sandbox) error {
		steps = append(steps, "stopRuntime")
		return nil
	})
	m.SetTeardownNetForTest(func(*config.Config, state.Sandbox) error {
		steps = append(steps, "teardownNet")
		return nil
	})
	m.SetRemoveJailForTest(func(*config.Config, state.Sandbox) error {
		steps = append(steps, "removeJail")
		return nil
	})
	m.SetRemoveAcctForTest(func(name string) error {
		steps = append(steps, "removeAcct:"+name)
		return nil
	})
	return &steps
}

func TestDestroyIdempotentAndRemovesState(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	steps := stubDestroyOK(t, m)

	sb, err := m.Claim("worker-1", 1, 256)
	if err != nil {
		t.Fatalf("Claim() = %v", err)
	}
	dir := filepath.Join(cfg.Firecracker.SandboxDirectory, sb.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := m.Destroy(sb); err != nil {
		t.Fatalf("Destroy() = %v", err)
	}
	if _, err := state.New(cfg.State.Directory).Load(sb.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("state after Destroy: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("storage dir still present")
	}

	wantAcct := "removeAcct:" + allocate.AccountName(sb.Slot)
	foundAcct := false
	for _, s := range *steps {
		if s == wantAcct {
			foundAcct = true
		}
	}
	if !foundAcct {
		t.Fatalf("steps = %v, want %s", *steps, wantAcct)
	}

	*steps = nil
	if err := m.Destroy(sb); err != nil {
		t.Fatalf("second Destroy() = %v", err)
	}
	if err := m.DestroyByID(sb.ID); err != nil {
		t.Fatalf("DestroyByID missing = %v", err)
	}
}

func TestDestroyLeavesFailedStateOnCleanupError(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	stubDestroyOK(t, m)
	m.SetTeardownNetForTest(func(*config.Config, state.Sandbox) error {
		return fmt.Errorf("boom")
	})

	sb, err := m.Claim("worker-1", 1, 256)
	if err != nil {
		t.Fatalf("Claim() = %v", err)
	}
	if err := m.Destroy(sb); err == nil {
		t.Fatal("Destroy() = nil, want error")
	}

	got, err := state.New(cfg.State.Directory).Load(sb.ID)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if got.ObservedState != state.ObservedFailed {
		t.Fatalf("ObservedState = %s, want failed", got.ObservedState)
	}
	if got.DesiredState != state.DesiredDeleted {
		t.Fatalf("DesiredState = %s, want deleted", got.DesiredState)
	}
	if got.LastError == "" {
		t.Fatal("LastError empty")
	}

	// Slot still claimed: next claim must not reuse it.
	other, err := m.Claim("worker-2", 1, 256)
	if err != nil {
		t.Fatalf("Claim(worker-2) = %v", err)
	}
	if other.Slot == sb.Slot {
		t.Fatalf("slot %d reused while failed sandbox still present", sb.Slot)
	}
}

func TestDestroyStopsUnitsBeforeNetTeardown(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	steps := stubDestroyOK(t, m)

	sb, err := m.Claim("worker-1", 1, 256)
	if err != nil {
		t.Fatalf("Claim() = %v", err)
	}
	if err := m.Destroy(sb); err != nil {
		t.Fatalf("Destroy() = %v", err)
	}

	order := *steps
	idx := func(prefix string) int {
		for i, s := range order {
			if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
				return i
			}
		}
		return -1
	}
	iUnit := idx("stopUnit:knaller@" + sb.ID)
	iRuntime := idx("stopRuntime")
	iNetUnit := idx("stopUnit:knaller-network@" + sb.ID)
	iTeardown := idx("teardownNet")
	if iUnit < 0 || iRuntime < 0 || iNetUnit < 0 || iTeardown < 0 {
		t.Fatalf("missing steps: %v", order)
	}
	if !(iUnit < iRuntime && iRuntime < iNetUnit && iNetUnit < iTeardown) {
		t.Fatalf("bad order: %v", order)
	}
}
