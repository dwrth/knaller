package jailer_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/jailer"
	"github.com/dwrth/knaller/internal/state"
)

func testConfig() *config.Config {
	return &config.Config{
		Jailer: config.Jailer{
			BaseDir:  "/srv/jailer",
			UidStart: 12000,
			GidStart: 12000,
		},
		Firecracker: config.Firecracker{
			SandboxDirectory: "/var/lib/knaller/vms",
		},
	}
}

func testSandbox() state.Sandbox {
	return state.Sandbox{
		ID:        "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Slot:      1,
		UID:       12001,
		GID:       12001,
		Namespace: "kn-sandbox-0001",
	}
}

func TestResolve(t *testing.T) {
	in, err := jailer.Resolve(testConfig(), testSandbox())
	if err != nil {
		t.Fatal(err)
	}

	id := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
	want := jailer.Inputs{
		ID:            id,
		UID:           12001,
		GID:           12001,
		NetNSPath:     "/run/netns/kn-sandbox-0001",
		ChrootBaseDir: "/srv/jailer",
		JailDir:       "/srv/jailer/firecracker/" + id,
		RootDir:       "/srv/jailer/firecracker/" + id + "/root",
		SandboxDir:    "/var/lib/knaller/vms/" + id,
		Kernel:        "/var/lib/knaller/vms/" + id + "/vmlinux",
		Rootfs:        "/var/lib/knaller/vms/" + id + "/rootfs.ext4",
		Config:        "/var/lib/knaller/vms/" + id + "/config.json",
		APISock:       "/srv/jailer/firecracker/" + id + "/root/run/firecracker.socket",
		PidFile:       "/srv/jailer/firecracker/" + id + "/root/firecracker.pid",
	}
	if in != want {
		t.Fatalf("Resolve() = %+v, want %+v", in, want)
	}
}

func TestResolveErrors(t *testing.T) {
	cfg := testConfig()
	sb := testSandbox()

	cases := []struct {
		name string
		cfg  *config.Config
		sb   state.Sandbox
	}{
		{"empty id", cfg, state.Sandbox{Namespace: "kn-sandbox-0001"}},
		{"empty namespace", cfg, state.Sandbox{ID: "x"}},
		{"empty base_dir", &config.Config{Firecracker: cfg.Firecracker}, sb},
		{"empty sandbox_directory", &config.Config{Jailer: cfg.Jailer}, sb},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := jailer.Resolve(tc.cfg, tc.sb); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestResolveID(t *testing.T) {
	dir := t.TempDir()
	store := state.New(dir)
	sb := testSandbox()
	sb.DesiredState = state.DesiredStopped
	sb.ObservedState = state.ObservedRequested
	if err := store.Create(sb); err != nil {
		t.Fatal(err)
	}

	in, err := jailer.ResolveID(testConfig(), store, sb.ID)
	if err != nil {
		t.Fatal(err)
	}
	if in.ID != sb.ID || in.UID != sb.UID || in.NetNSPath != "/run/netns/"+sb.Namespace {
		t.Fatalf("ResolveID() = %+v", in)
	}

	_, err = jailer.ResolveID(testConfig(), store, "missing")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing id error = %v, want ErrNotExist", err)
	}
}

func TestResolveDoesNotRequireArtifacts(t *testing.T) {
	// Resolve must succeed even when durable files are absent.
	cfg := testConfig()
	cfg.Firecracker.SandboxDirectory = filepath.Join(t.TempDir(), "vms")
	if _, err := jailer.Resolve(cfg, testSandbox()); err != nil {
		t.Fatal(err)
	}
}
