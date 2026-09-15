package allocate

import (
	"fmt"

	"github.com/dwrth/knaller/internal/state"
)

const maxSlot = 255

// NextSlot returns the lowest unused slot number among existing sandboxes.
//
// Sandboxes with Slot >= 1 occupy a slot; Slot 0 is treated as unset. Pass existing
// from persisted state so deleted sandboxes release their slot for reuse.
func NextSlot(existing []state.Sandbox) (int, error) {
	used := make(map[int]struct{}, len(existing))
	for _, sandbox := range existing {
		if sandbox.Slot >= 1 {
			used[sandbox.Slot] = struct{}{}
		}
	}

	for n := 1; ; n++ {
		if n > maxSlot {
			return 0, fmt.Errorf("allocate: no slots available (max %d)", maxSlot)
		}
		if _, ok := used[n]; !ok {
			return n, nil
		}
	}
}

// NextSlotFromStore retrieves all sandboxes from the provided store and returns the lowest unused slot number.
// It returns an error if listing sandboxes from the store fails.
func NextSlotFromStore(store *state.Store) (int, error) {
	existing, err := store.List()
	if err != nil {
		return 0, err
	}
	return NextSlot(existing)
}
