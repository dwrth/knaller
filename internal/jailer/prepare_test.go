package jailer_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/jailer"
	"github.com/dwrth/knaller/internal/state"
)

func prepareEnv(t *testing.T) (*config.Config, state.Sandbox) {
	t.Helper()
	root := t.TempDir()
	id := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	cfg := &config.Config{
		Jailer:      config.Jailer{BaseDir: filepath.Join(root, "jailer")},
		Firecracker: config.Firecracker{SandboxDirectory: filepath.Join(root, "vms")},
	}
	sb := state.Sandbox{
		ID:        id,
		UID:       12001,
		GID:       12001,
		Namespace: "kn-sandbox-0001",
	}
	dir := filepath.Join(cfg.Firecracker.SandboxDirectory, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"vmlinux", "rootfs.ext4", "config.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return cfg, sb
}

func TestPrepare(t *testing.T) {
	cfg, sb := prepareEnv(t)
	if err := jailer.Prepare(cfg, sb); err != nil {
		t.Fatal(err)
	}
	in, err := jailer.Resolve(cfg, sb)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(in.RootDir, "vmlinux"),
		filepath.Join(in.RootDir, "rootfs.ext4"),
		filepath.Join(in.RootDir, "config.json"),
	} {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if !fi.Mode().IsRegular() {
			t.Fatalf("%s is not a regular file", path)
		}
	}

	// Idempotent rebuild.
	if err := jailer.Prepare(cfg, sb); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareMissingArtifact(t *testing.T) {
	cfg, sb := prepareEnv(t)
	in, err := jailer.Resolve(cfg, sb)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(in.Config); err != nil {
		t.Fatal(err)
	}
	if err := jailer.Prepare(cfg, sb); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(in.JailDir); !os.IsNotExist(err) {
		t.Fatalf("jail dir should not exist before failed prepare, err=%v", err)
	}
}

func TestPrepareRefusesRunning(t *testing.T) {
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
	err = jailer.Prepare(cfg, sb)
	if !errors.Is(err, jailer.ErrRunning) {
		t.Fatalf("error = %v, want ErrRunning", err)
	}
	// Jail must remain (refuse happened before wipe).
	if _, err := os.Stat(in.RootDir); err != nil {
		t.Fatalf("jail should remain: %v", err)
	}
}

func TestPrepareAllowsStalePidfile(t *testing.T) {
	cfg, sb := prepareEnv(t)
	if err := jailer.Prepare(cfg, sb); err != nil {
		t.Fatal(err)
	}
	in, err := jailer.Resolve(cfg, sb)
	if err != nil {
		t.Fatal(err)
	}
	// PID 1 is not our firecracker; use a pid that is almost certainly dead.
	if err := os.WriteFile(in.PidFile, []byte("2147483647"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := jailer.Prepare(cfg, sb); err != nil {
		t.Fatal(err)
	}
}
