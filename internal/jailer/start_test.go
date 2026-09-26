package jailer_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/dwrth/knaller/internal/jailer"
)

func TestArgv(t *testing.T) {
	in := jailer.Inputs{
		ID:            "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		UID:           12001,
		GID:           12001,
		NetNSPath:     "/run/netns/kn-sandbox-0001",
		ChrootBaseDir: "/srv/jailer",
	}
	got := jailer.Argv(in)
	want := []string{
		"/usr/local/bin/jailer",
		"--id", "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"--exec-file", "/usr/local/bin/firecracker",
		"--uid", "12001",
		"--gid", "12001",
		"--chroot-base-dir", "/srv/jailer",
		"--netns", "/run/netns/kn-sandbox-0001",
		"--cgroup-version", "2",
		"--resource-limit", "no-file=1024",
		"--",
		"--api-sock", "/run/firecracker.socket",
		"--config-file", "/config.json",
	}
	if len(got) != len(want) {
		t.Fatalf("Argv len = %d, want %d\n got %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Argv[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStartFailsWhenNetnsMissing(t *testing.T) {
	cfg, sb := prepareEnv(t)
	if err := jailer.Prepare(cfg, sb); err != nil {
		t.Fatal(err)
	}
	// NetNSPath resolves under /run/netns, which will not exist for this namespace.
	err := jailer.Start(cfg, sb)
	if err == nil {
		t.Fatal("expected error before exec")
	}
	if errors.Is(err, jailer.ErrRunning) {
		t.Fatalf("unexpected ErrRunning: %v", err)
	}
}

func TestStartFailsWhenJailMissing(t *testing.T) {
	cfg, sb := prepareEnv(t)
	// Durable artifacts exist (prepareEnv), but jail was never prepared.
	err := jailer.Start(cfg, sb)
	if err == nil {
		t.Fatal("expected error before exec")
	}
}

func TestStartFailsWhenRunning(t *testing.T) {
	cfg, sb := prepareEnv(t)
	if err := jailer.Prepare(cfg, sb); err != nil {
		t.Fatal(err)
	}
	in, err := jailer.Resolve(cfg, sb)
	if err != nil {
		t.Fatal(err)
	}
	// Point NetNSPath check out of the way by also needing running check first.
	// ensureStopped runs before netns check; live pidfile is enough.
	if err := os.MkdirAll(filepath.Dir(in.PidFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in.PidFile, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	err = jailer.Start(cfg, sb)
	if !errors.Is(err, jailer.ErrRunning) {
		t.Fatalf("error = %v, want ErrRunning", err)
	}
}
