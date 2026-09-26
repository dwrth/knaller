package jailer_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/dwrth/knaller/internal/jailer"
)

func TestRemove(t *testing.T) {
	cfg, sb := prepareEnv(t)
	if err := jailer.Prepare(cfg, sb); err != nil {
		t.Fatal(err)
	}
	in, err := jailer.Resolve(cfg, sb)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(in.JailDir); err != nil {
		t.Fatal(err)
	}
	if err := jailer.Remove(cfg, sb); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(in.JailDir); !os.IsNotExist(err) {
		t.Fatalf("jail should be gone, err=%v", err)
	}
	// Idempotent.
	if err := jailer.Remove(cfg, sb); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveRefusesRunning(t *testing.T) {
	cfg, sb := prepareEnv(t)
	if err := jailer.Prepare(cfg, sb); err != nil {
		t.Fatal(err)
	}
	in, err := jailer.Resolve(cfg, sb)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in.PidFile, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	err = jailer.Remove(cfg, sb)
	if !errors.Is(err, jailer.ErrRunning) {
		t.Fatalf("error = %v, want ErrRunning", err)
	}
	if _, err := os.Stat(filepath.Join(in.RootDir, "vmlinux")); err != nil {
		t.Fatalf("jail should remain: %v", err)
	}
}
