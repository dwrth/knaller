package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dwrth/knaller/internal/state"
)

func writeTestConfig(t *testing.T, stateDir string) string {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, "base.ext4")
	kernel := filepath.Join(root, "vmlinux")
	for _, path := range []string{base, kernel} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, "config.yaml")
	content := `
storage:
  base_rootfs: ` + base + `
  kernel: ` + kernel + `
scheduler:
  host_memory_reserve_mib: 768
  cpu_overcommit_ratio: 2.0
  minimum_sandbox_memory_mib: 128
  maximum_sandbox_memory_mib: 8192
  maximum_sandbox_vcpus: 32
network:
  guest_cidr: 172.16.0.0/16
  transit_cidr: 10.200.0.0/16
jailer:
  base_dir: /tmp/jailer
  uid_start: 12000
  gid_start: 12000
state:
  directory: ` + stateDir + `
firecracker:
  sandbox_directory: /tmp/vms
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNetworkMissingSubcommand(t *testing.T) {
	if code := run([]string{"network"}); code != 2 {
		t.Fatalf("run(network) = %d, want 2", code)
	}
}

func TestNetworkSetupRequiresID(t *testing.T) {
	cfg := writeTestConfig(t, t.TempDir())
	if code := run([]string{"network", "setup", "--config", cfg}); code != 2 {
		t.Fatalf("run(setup without id) = %d, want 2", code)
	}
}

func TestNetworkSetupMissingState(t *testing.T) {
	stateDir := t.TempDir()
	cfg := writeTestConfig(t, stateDir)
	if code := run([]string{"network", "setup", "--config", cfg, "nope"}); code != 1 {
		t.Fatalf("run(setup missing state) = %d, want 1", code)
	}
}

func TestNetworkStatusPrintsReport(t *testing.T) {
	stateDir := t.TempDir()
	cfgPath := writeTestConfig(t, stateDir)
	store := state.New(stateDir)
	sb := state.Sandbox{
		ID:            "01HTESTNETWORKSTATUS00001",
		Name:          "status-test",
		Slot:          1,
		UID:           12001,
		GID:           12001,
		VCPUs:         1,
		MemoryMiB:     256,
		GuestIP:       "172.16.1.2",
		GatewayIP:     "172.16.1.1",
		GuestSubnet:   "172.16.1.0/30",
		TransitHostIP: "10.200.1.1",
		TransitNSIP:   "10.200.1.2",
		TransitSubnet: "10.200.1.0/30",
		Namespace:     "kn-sandbox-0001",
		HostVeth:      "kn1-host",
		NSVeth:        "kn1-ns",
		TAP:           "tap0",
		DesiredState:  state.DesiredStopped,
		ObservedState: state.ObservedRequested,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.Create(sb); err != nil {
		t.Fatal(err)
	}

	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := run([]string{"network", "status", "--config", cfgPath, sb.ID})
	_ = w.Close()
	os.Stdout = stdout
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	_ = r.Close()
	out := string(buf[:n])

	if code != 0 {
		t.Fatalf("run(status) = %d, want 0; out=%q", code, out)
	}
	for _, want := range []string{"netns", "host_veth", "guest_route", "absent"} {
		if !strings.Contains(out, want) {
			t.Errorf("status output missing %q:\n%s", want, out)
		}
	}
}
