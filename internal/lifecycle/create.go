package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dwrth/knaller/internal/allocate"
	"github.com/dwrth/knaller/internal/state"
	"github.com/dwrth/knaller/internal/storage"
	"github.com/dwrth/knaller/internal/vm"
)

// Create claims a sandbox, provisions host resources, starts it via systemd,
// and marks observed state running. On any provision failure after claim,
// Destroy is invoked for rollback.
func (m *Manager) Create(name string, vcpus, memoryMiB int) (state.Sandbox, error) {
	sb, err := m.Claim(name, vcpus, memoryMiB)
	if err != nil {
		return state.Sandbox{}, err
	}

	if err := m.provision(sb); err != nil {
		if destroyErr := m.Destroy(sb); destroyErr != nil {
			return state.Sandbox{}, fmt.Errorf("%w (rollback failed: %v)", err, destroyErr)
		}
		return state.Sandbox{}, err
	}

	return m.markRunning(sb)
}

func (m *Manager) provision(sb state.Sandbox) error {
	acct := allocate.AccountName(sb.Slot)
	if err := m.ensureAcct(acct, sb.UID, sb.GID); err != nil {
		return fmt.Errorf("lifecycle: account: %w", err)
	}
	if err := m.provisionStorage(sb); err != nil {
		return fmt.Errorf("lifecycle: storage: %w", err)
	}
	if err := m.setObserved(sb.ID, state.ObservedPrepared); err != nil {
		return err
	}
	if err := m.ensureHost(m.cfg); err != nil {
		return fmt.Errorf("lifecycle: host network: %w", err)
	}
	if err := m.setObserved(sb.ID, state.ObservedStarting); err != nil {
		return err
	}

	unit := "knaller@" + sb.ID + ".service"
	if err := m.startUnit(unit); err != nil {
		return fmt.Errorf("lifecycle: start %s: %w", unit, err)
	}
	if err := m.waitReady(m.cfg, sb); err != nil {
		return fmt.Errorf("lifecycle: ready: %w", err)
	}
	return nil
}

func (m *Manager) setObserved(id string, observed state.ObservedState) error {
	sb, err := m.store.Load(id)
	if err != nil {
		return fmt.Errorf("lifecycle: load for state update: %w", err)
	}
	sb.ObservedState = observed
	sb.LastError = ""
	if err := m.store.Save(sb); err != nil {
		return fmt.Errorf("lifecycle: save observed state: %w", err)
	}
	return nil
}

func (m *Manager) markRunning(sb state.Sandbox) (state.Sandbox, error) {
	unlock, err := m.lock.TryLock()
	if err != nil {
		return state.Sandbox{}, err
	}
	defer func() { _ = unlock() }()

	cur, err := m.store.Load(sb.ID)
	if err != nil {
		return state.Sandbox{}, fmt.Errorf("lifecycle: load: %w", err)
	}
	cur.DesiredState = state.DesiredRunning
	cur.ObservedState = state.ObservedRunning
	cur.LastError = ""
	if err := m.store.Save(cur); err != nil {
		return state.Sandbox{}, fmt.Errorf("lifecycle: mark running: %w", err)
	}
	return cur, nil
}

func defaultProvisionStorage(store *storage.Manager) func(state.Sandbox) error {
	return func(sb state.Sandbox) error {
		if err := store.CloneBaseRootfs(sb.ID); err != nil {
			return err
		}
		if err := store.CustomizeRootfs(sb); err != nil {
			return err
		}
		if err := store.InstallKernel(sb.ID); err != nil {
			return err
		}
		if err := writeFirecrackerConfig(store, sb); err != nil {
			return err
		}
		return chownSandboxArtifacts(store, sb)
	}
}

func writeFirecrackerConfig(store *storage.Manager, sb state.Sandbox) error {
	fc, err := vm.GenerateFirecrackerConfig(sb)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return fmt.Errorf("lifecycle: marshal firecracker config: %w", err)
	}
	path := store.Config(sb.ID)
	if err := os.WriteFile(path, data, 0o440); err != nil {
		return fmt.Errorf("lifecycle: write firecracker config: %w", err)
	}
	return nil
}

func chownSandboxArtifacts(store *storage.Manager, sb state.Sandbox) error {
	rootfs := store.Rootfs(sb.ID)
	if err := os.Chown(rootfs, sb.UID, sb.GID); err != nil {
		return fmt.Errorf("lifecycle: chown rootfs: %w", err)
	}
	if err := os.Chmod(rootfs, 0o600); err != nil {
		return fmt.Errorf("lifecycle: chmod rootfs: %w", err)
	}
	for _, path := range []string{store.Kernel(sb.ID), store.Config(sb.ID)} {
		if err := os.Chown(path, 0, sb.GID); err != nil {
			return fmt.Errorf("lifecycle: chown %s: %w", path, err)
		}
		if err := os.Chmod(path, 0o440); err != nil {
			return fmt.Errorf("lifecycle: chmod %s: %w", path, err)
		}
	}
	return nil
}
