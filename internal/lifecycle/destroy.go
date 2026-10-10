package lifecycle

import (
	"errors"
	"fmt"
	"os"

	"github.com/dwrth/knaller/internal/allocate"
	"github.com/dwrth/knaller/internal/state"
)

// Destroy tears down host resources for sandbox and removes its state record.
// Missing resources are ignored. If cleanup fails partway, state is left as
// failed with LastError and the slot remains claimed.
// The node mutation lock is held for the whole call.
func (m *Manager) Destroy(sandbox state.Sandbox) error {
	unlock, err := m.lock.TryLock()
	if err != nil {
		return err
	}
	defer func() { _ = unlock() }()

	return m.destroyLocked(sandbox)
}

// DestroyByID loads sandbox state and destroys it. Missing state is success.
func (m *Manager) DestroyByID(id string) error {
	unlock, err := m.lock.TryLock()
	if err != nil {
		return err
	}
	defer func() { _ = unlock() }()

	sb, err := m.store.Load(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("lifecycle: load: %w", err)
	}
	return m.destroyLocked(sb)
}

func (m *Manager) destroyLocked(sandbox state.Sandbox) error {
	sandbox.DesiredState = state.DesiredDeleted
	sandbox.ObservedState = state.ObservedDeleting
	sandbox.LastError = ""
	// Best-effort: cleanup proceeds even if the record cannot be updated.
	_ = m.store.Save(sandbox)

	var first error
	note := func(err error) {
		if err != nil && first == nil {
			first = err
		}
	}

	note(m.stopUnit("knaller@" + sandbox.ID + ".service"))
	note(m.stopRuntime(m.cfg, sandbox))
	note(m.stopUnit("knaller-network@" + sandbox.ID + ".service"))
	note(m.teardownNet(m.cfg, sandbox))
	note(m.removeJail(m.cfg, sandbox))
	note(m.storage.DeleteSandboxStorage(sandbox.ID))
	note(m.removeAcct(allocate.AccountName(sandbox.Slot)))

	if first != nil {
		sandbox.ObservedState = state.ObservedFailed
		sandbox.LastError = first.Error()
		_ = m.store.Save(sandbox)
		return first
	}

	if err := m.store.Delete(sandbox.ID); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("lifecycle: delete state: %w", err)
	}
	return nil
}
