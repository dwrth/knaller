package lifecycle

import (
	"fmt"

	"github.com/dwrth/knaller/internal/allocate"
	"github.com/dwrth/knaller/internal/scheduler"
	"github.com/dwrth/knaller/internal/state"
)

// Claim admits and allocates a sandbox, then persists it as provisioning.
// The node mutation lock is held for the whole call.
func (m *Manager) Claim(name string, vcpus, memoryMiB int) (state.Sandbox, error) {
	unlock, err := m.lock.TryLock()
	if err != nil {
		return state.Sandbox{}, err
	}
	defer func() { _ = unlock() }()

	existing, err := m.store.List()
	if err != nil {
		return state.Sandbox{}, fmt.Errorf("lifecycle: list sandboxes: %w", err)
	}
	for _, sb := range existing {
		if sb.Name == name {
			return state.Sandbox{}, fmt.Errorf("%w: %s", ErrNameTaken, name)
		}
	}

	cap, err := m.collect(m.cfg)
	if err != nil {
		return state.Sandbox{}, fmt.Errorf("lifecycle: capacity: %w", err)
	}
	if err := scheduler.Admit(m.cfg, cap, scheduler.Request{VCPUs: vcpus, MemoryMiB: memoryMiB}); err != nil {
		return state.Sandbox{}, err
	}

	sb, err := allocate.Build(existing, m.cfg, name, vcpus, memoryMiB)
	if err != nil {
		return state.Sandbox{}, err
	}
	sb.DesiredState = state.DesiredRunning
	sb.ObservedState = state.ObservedProvisioning
	sb.LastError = ""

	if err := m.store.Create(sb); err != nil {
		return state.Sandbox{}, fmt.Errorf("lifecycle: persist: %w", err)
	}
	return sb, nil
}
