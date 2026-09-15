package allocate_test

import (
	"testing"

	"github.com/dwrth/knaller/internal/allocate"
	"github.com/dwrth/knaller/internal/state"
	"github.com/oklog/ulid/v2"
)

func TestNextSlot(t *testing.T) {
	uniqueID := func() string { return ulid.Make().String() }
	tests := []struct {
		name     string
		existing []state.Sandbox
		want     int
		wantErr  bool
	}{

		{
			name: "take empty slot inbetween",
			existing: []state.Sandbox{
				{Name: "test1", ID: uniqueID(), Slot: 1},
				{Name: "test3", ID: uniqueID(), Slot: 3},
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "take empty 0 slot inbetween",
			existing: []state.Sandbox{
				{Name: "test1", ID: uniqueID(), Slot: 1},
				{Name: "test2", ID: uniqueID(), Slot: 0},
				{Name: "test3", ID: uniqueID(), Slot: 3},
			},
			want:    2,
			wantErr: false,
		},
		{
			name:    "no existing slots",
			want:    1,
			wantErr: false,
		},
		{
			name:     "no available slots",
			existing: buildMaxSandboxes(),
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := allocate.NextSlot(tt.existing)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("NextSlot() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("NextSlot() succeeded unexpectedly")
			}
			if got != tt.want {
				t.Errorf("NextSlot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNextSlotFromStore_reusesAfterDelete(t *testing.T) {
	store := state.New(t.TempDir())
	s1 := persistedSandbox(t, "a", 1)
	s2 := persistedSandbox(t, "b", 2)
	s3 := persistedSandbox(t, "c", 3)
	for _, sb := range []state.Sandbox{s1, s2, s3} {
		if err := store.Create(sb); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Delete(s2.ID); err != nil {
		t.Fatal(err)
	}

	got, err := allocate.NextSlotFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("NextSlotFromStore() = %d, want 2", got)
	}
}

func TestNextSlotFromStore_desiredDeletedStillBlocks(t *testing.T) {
	store := state.New(t.TempDir())
	sb := persistedSandbox(t, "doomed", 2)
	sb.DesiredState = state.DesiredDeleted
	if err := store.Create(sb); err != nil {
		t.Fatal(err)
	}

	got, err := allocate.NextSlotFromStore(store)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("NextSlotFromStore() = %d, want 1 (slot 2 still held)", got)
	}
}

func persistedSandbox(t *testing.T, name string, slot int) state.Sandbox {
	t.Helper()
	return state.Sandbox{
		ID:            ulid.Make().String(),
		Name:          name,
		Slot:          slot,
		DesiredState:  state.DesiredStopped,
		ObservedState: state.ObservedStopped,
	}
}

func buildMaxSandboxes() []state.Sandbox {
	existing := make([]state.Sandbox, 255)
	for i := range existing {
		existing[i] = state.Sandbox{Name: "test", Slot: i + 1}
	}

	return existing
}
