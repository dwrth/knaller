package lifecycle_test

import (
	"errors"
	"testing"

	"github.com/dwrth/knaller/internal/lifecycle"
)

func TestLookupByNameAndID(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	stubCreateOK(t, m)

	sb, err := m.Create("worker-1", 1, 256)
	if err != nil {
		t.Fatalf("Create() = %v", err)
	}

	byName, err := m.Lookup("worker-1")
	if err != nil || byName.ID != sb.ID {
		t.Fatalf("Lookup(name) = %+v, %v", byName, err)
	}
	byID, err := m.Lookup(sb.ID)
	if err != nil || byID.Name != "worker-1" {
		t.Fatalf("Lookup(id) = %+v, %v", byID, err)
	}
	_, err = m.Lookup("missing")
	if !errors.Is(err, lifecycle.ErrNotFound) {
		t.Fatalf("Lookup(missing) = %v, want ErrNotFound", err)
	}
}

func TestDestroyRef(t *testing.T) {
	cfg := testCfg(t)
	m := lifecycle.New(cfg)
	stubCapacity(t, m)
	stubCreateOK(t, m)

	if _, err := m.Create("worker-1", 1, 256); err != nil {
		t.Fatalf("Create() = %v", err)
	}
	if err := m.DestroyRef("worker-1"); err != nil {
		t.Fatalf("DestroyRef() = %v", err)
	}
	assertNoSandboxState(t, cfg.State.Directory)
}
