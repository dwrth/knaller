package lifecycle_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/lifecycle"
	"github.com/dwrth/knaller/internal/state"
)

// Forced failures at each provision boundary must either roll back
// cleanly (no state, slot reusable) or leave a diagnosable failed
// record when rollback itself fails, never silent debris.

func TestCreateFailsAtEachStepRollsBackCleanly(t *testing.T) {
	cases := []struct {
		name string
		fail func(*lifecycle.Manager)
		want string
	}{
		{
			name: "account",
			fail: func(m *lifecycle.Manager) {
				m.SetEnsureAcctForTest(func(string, int, int) error {
					return fmt.Errorf("account denied")
				})
			},
			want: "account denied",
		},
		{
			name: "storage",
			fail: func(m *lifecycle.Manager) {
				m.SetProvisionStorageForTest(func(state.Sandbox) error {
					return fmt.Errorf("disk full")
				})
			},
			want: "disk full",
		},
		{
			name: "host_network",
			fail: func(m *lifecycle.Manager) {
				m.SetEnsureHostForTest(func(*config.Config) error {
					return fmt.Errorf("nft failed")
				})
			},
			want: "nft failed",
		},
		{
			name: "start_unit",
			fail: func(m *lifecycle.Manager) {
				m.SetStartUnitForTest(func(string) error {
					return fmt.Errorf("unit failed")
				})
			},
			want: "unit failed",
		},
		{
			name: "wait_ready",
			fail: func(m *lifecycle.Manager) {
				m.SetWaitReadyForTest(func(*config.Config, state.Sandbox) error {
					return fmt.Errorf("not active")
				})
			},
			want: "not active",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testCfg(t)
			m := lifecycle.New(cfg)
			stubCapacity(t, m)
			steps := stubCreateOK(t, m)
			tc.fail(m)

			_, err := m.Create("worker-1", 1, 256)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Create() = %v, want %q", err, tc.want)
			}
			assertNoSandboxState(t, cfg.State.Directory)
			assertDestroyInvoked(t, *steps)

			// Slot and name reusable after clean rollback.
			stubCreateOK(t, m)
			sb, err := m.Create("worker-1", 1, 256)
			if err != nil {
				t.Fatalf("Create after rollback = %v", err)
			}
			if sb.Slot != 1 {
				t.Fatalf("slot = %d, want 1", sb.Slot)
			}
		})
	}
}

func TestCreateFailureRemovesStorageDebris(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	stubCreateOK(t, m)

	var storageDir string
	m.SetProvisionStorageForTest(func(sb state.Sandbox) error {
		storageDir = filepath.Join(cfg.Firecracker.SandboxDirectory, sb.ID)
		if err := os.MkdirAll(storageDir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(storageDir, "rootfs.ext4"), []byte("x"), 0o644)
	})
	m.SetStartUnitForTest(func(string) error {
		return fmt.Errorf("boom")
	})

	_, err := m.Create("worker-1", 1, 256)
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Create() = %v, want boom", err)
	}
	assertNoSandboxState(t, cfg.State.Directory)
	if storageDir == "" {
		t.Fatal("storage dir was not recorded")
	}
	if _, err := os.Stat(storageDir); !os.IsNotExist(err) {
		t.Fatalf("storage debris still present at %s: %v", storageDir, err)
	}
}

func TestCreateRollbackFailureLeavesFailedState(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	stubCreateOK(t, m)

	m.SetProvisionStorageForTest(func(state.Sandbox) error {
		return fmt.Errorf("clone failed")
	})
	m.SetTeardownNetForTest(func(*config.Config, state.Sandbox) error {
		return fmt.Errorf("teardown stuck")
	})

	_, err := m.Create("worker-1", 1, 256)
	if err == nil {
		t.Fatal("Create() = nil, want error")
	}
	if !strings.Contains(err.Error(), "clone failed") {
		t.Fatalf("Create() = %v, want clone failed", err)
	}
	if !strings.Contains(err.Error(), "rollback failed") {
		t.Fatalf("Create() = %v, want rollback failed", err)
	}

	list, listErr := state.New(cfg.State.Directory).List()
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(list) != 1 {
		t.Fatalf("sandboxes = %d, want 1 failed record", len(list))
	}
	got := list[0]
	if got.ObservedState != state.ObservedFailed {
		t.Fatalf("ObservedState = %s, want failed", got.ObservedState)
	}
	if got.DesiredState != state.DesiredDeleted {
		t.Fatalf("DesiredState = %s, want deleted", got.DesiredState)
	}
	if got.LastError == "" {
		t.Fatal("LastError empty")
	}
	if got.Slot != 1 {
		t.Fatalf("slot = %d, want 1", got.Slot)
	}

	// Slot stays claimed so the next create cannot collide.
	stubCreateOK(t, m)
	other, err := m.Create("worker-2", 1, 256)
	if err != nil {
		t.Fatalf("Create(worker-2) = %v", err)
	}
	if other.Slot == got.Slot {
		t.Fatalf("slot %d reused while failed sandbox still present", got.Slot)
	}
}

func TestCreateDestroyCycleReusesSlot(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	stubCreateOK(t, m)

	sb, err := m.Create("worker-1", 2, 512)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}
	id := sb.ID
	if err := m.DestroyRef("worker-1"); err != nil {
		t.Fatalf("DestroyRef() = %v", err)
	}
	assertNoSandboxState(t, cfg.State.Directory)

	again, err := m.Create("worker-1", 2, 512)
	if err != nil {
		t.Fatalf("Create after destroy = %v", err)
	}
	if again.Slot != 1 {
		t.Fatalf("slot = %d, want 1", again.Slot)
	}
	if again.ID == id {
		t.Fatal("sandbox ID reused after destroy; want a new identity")
	}
}

func assertDestroyInvoked(t *testing.T, steps []string) {
	t.Helper()
	needles := []string{"stopRuntime", "teardownNet", "removeJail", "removeAcct:"}
	for _, n := range needles {
		found := false
		for _, s := range steps {
			if strings.Contains(s, n) || s == n {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("destroy steps %v missing %q", steps, n)
		}
	}
}
