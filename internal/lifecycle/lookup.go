package lifecycle

import (
	"errors"
	"fmt"

	"github.com/dwrth/knaller/internal/state"
)

// ErrNotFound is returned when Lookup cannot resolve a sandbox name or ID.
var ErrNotFound = errors.New("lifecycle: sandbox not found")

// Lookup finds a sandbox by exact name or ID. Name matches take precedence.
func (m *Manager) Lookup(ref string) (state.Sandbox, error) {
	if ref == "" {
		return state.Sandbox{}, fmt.Errorf("%w: empty reference", ErrNotFound)
	}
	list, err := m.store.List()
	if err != nil {
		return state.Sandbox{}, fmt.Errorf("lifecycle: list: %w", err)
	}
	var byID *state.Sandbox
	for i := range list {
		sb := &list[i]
		if sb.Name == ref {
			return *sb, nil
		}
		if sb.ID == ref {
			byID = sb
		}
	}
	if byID != nil {
		return *byID, nil
	}
	return state.Sandbox{}, fmt.Errorf("%w: %s", ErrNotFound, ref)
}

// DestroyRef looks up a sandbox by name or ID and destroys it.
func (m *Manager) DestroyRef(ref string) error {
	sb, err := m.Lookup(ref)
	if err != nil {
		return err
	}
	return m.Destroy(sb)
}
