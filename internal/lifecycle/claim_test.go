package lifecycle_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/dwrth/knaller/internal/capacity"
	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/lifecycle"
	"github.com/dwrth/knaller/internal/lock"
	"github.com/dwrth/knaller/internal/state"
)

func testCfg(t *testing.T) *config.Config {
	t.Helper()
	tmp := t.TempDir()
	return &config.Config{
		Storage: config.Storage{
			BaseRootfs: filepath.Join(tmp, "base.ext4"),
			Kernel:     filepath.Join(tmp, "vmlinux"),
		},
		Scheduler: config.Scheduler{
			HostMemoryReserveMiB:    256,
			CPUOvercommitRatio:      2.0,
			MinimumSandboxMemoryMiB: 128,
			MaximumSandboxMemoryMiB: 2048,
			MaximumSandboxVCPUs:     4,
		},
		Network: config.Network{
			GuestCidr:   "172.16.0.0/16",
			TransitCidr: "10.200.0.0/16",
		},
		Jailer: config.Jailer{
			BaseDir:  filepath.Join(tmp, "jailer"),
			UidStart: 12000,
			GidStart: 12000,
		},
		State: config.State{
			Directory: filepath.Join(tmp, "state"),
		},
		Firecracker: config.Firecracker{
			SandboxDirectory: filepath.Join(tmp, "vms"),
		},
	}
}

func stubCapacity(t *testing.T, m *lifecycle.Manager) {
	t.Helper()
	m.SetCollectForTest(func(*config.Config) (capacity.Capacity, error) {
		return capacity.Capacity{
			HostCPUs:           8,
			HostMemoryMiB:      16384,
			ReservedMemoryMiB:  256,
			CPUOvercommitRatio: 2.0,
		}, nil
	})
}

func TestClaimPersistsProvisioning(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)

	sb, err := m.Claim("worker-1", 2, 512)
	if err != nil {
		t.Fatalf("Claim() = %v", err)
	}
	if sb.ID == "" || sb.Slot != 1 {
		t.Fatalf("Claim() id/slot = %q/%d", sb.ID, sb.Slot)
	}
	if sb.DesiredState != state.DesiredRunning || sb.ObservedState != state.ObservedProvisioning {
		t.Fatalf("states = %s/%s", sb.DesiredState, sb.ObservedState)
	}

	got, err := state.New(cfg.State.Directory).Load(sb.ID)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if got.Name != "worker-1" || got.ObservedState != state.ObservedProvisioning {
		t.Fatalf("persisted = %+v", got)
	}
}

func TestClaimRejectsDuplicateName(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)

	if _, err := m.Claim("worker-1", 1, 256); err != nil {
		t.Fatalf("first Claim() = %v", err)
	}
	_, err := m.Claim("worker-1", 1, 256)
	if !errors.Is(err, lifecycle.ErrNameTaken) {
		t.Fatalf("second Claim() = %v, want ErrNameTaken", err)
	}
}

func TestClaimBusyWhenLocked(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)

	l := lock.New(filepath.Join(cfg.State.Directory, "knaller.lock"))
	unlock, err := l.TryLock()
	if err != nil {
		t.Fatalf("TryLock() = %v", err)
	}
	defer func() { _ = unlock() }()

	_, err = m.Claim("worker-1", 1, 256)
	if !errors.Is(err, lifecycle.ErrBusy) {
		t.Fatalf("Claim() = %v, want ErrBusy", err)
	}
}

func TestClaimAllocatesDistinctSlots(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)

	a, err := m.Claim("a", 1, 256)
	if err != nil {
		t.Fatalf("Claim(a) = %v", err)
	}
	b, err := m.Claim("b", 1, 256)
	if err != nil {
		t.Fatalf("Claim(b) = %v", err)
	}
	if a.Slot == b.Slot || a.ID == b.ID || a.UID == b.UID {
		t.Fatalf("expected distinct allocation: a=%+v b=%+v", a, b)
	}
}
