package network_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/jailer"
	"github.com/dwrth/knaller/internal/network"
	"github.com/dwrth/knaller/internal/state"
)

func validSandbox() state.Sandbox {
	return state.Sandbox{
		ID:            "01HTESTSANDBOX000000000000",
		Namespace:     "kn-sandbox-0001",
		HostVeth:      "kn1-host",
		NSVeth:        "kn1-ns",
		TAP:           "tap0",
		GuestSubnet:   "172.16.1.0/30",
		GatewayIP:     "172.16.1.1",
		TransitHostIP: "10.200.1.1",
		TransitNSIP:   "10.200.1.2",
		TransitSubnet: "10.200.1.0/30",
		UID:           12001,
		GID:           12001,
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		Jailer:      config.Jailer{BaseDir: filepath.Join(dir, "jail")},
		Firecracker: config.Firecracker{SandboxDirectory: filepath.Join(dir, "sandboxes")},
	}
}

func TestValidateRequiredFields(t *testing.T) {
	cfg := testConfig(t)
	restore := network.SetExecRunForTest(func(string, ...string) error { return nil })
	defer restore()

	tests := []struct {
		name string
		mut  func(*state.Sandbox)
		want string
	}{
		{name: "valid", mut: func(*state.Sandbox) {}, want: ""},
		{name: "missing id", mut: func(s *state.Sandbox) { s.ID = "" }, want: "id is required"},
		{name: "missing namespace", mut: func(s *state.Sandbox) { s.Namespace = "" }, want: "namespace is required"},
		{name: "missing host_veth", mut: func(s *state.Sandbox) { s.HostVeth = "" }, want: "host_veth is required"},
		{name: "missing ns_veth", mut: func(s *state.Sandbox) { s.NSVeth = "" }, want: "ns_veth is required"},
		{name: "missing tap", mut: func(s *state.Sandbox) { s.TAP = "" }, want: "tap is required"},
		{name: "missing guest_subnet", mut: func(s *state.Sandbox) { s.GuestSubnet = "" }, want: "guest_subnet is required"},
		{name: "missing gateway_ip", mut: func(s *state.Sandbox) { s.GatewayIP = "" }, want: "gateway_ip is required"},
		{name: "missing transit_host_ip", mut: func(s *state.Sandbox) { s.TransitHostIP = "" }, want: "transit_host_ip is required"},
		{name: "missing transit_ns_ip", mut: func(s *state.Sandbox) { s.TransitNSIP = "" }, want: "transit_ns_ip is required"},
		{name: "missing transit_subnet", mut: func(s *state.Sandbox) { s.TransitSubnet = "" }, want: "transit_subnet is required"},
		{name: "missing uid", mut: func(s *state.Sandbox) { s.UID = 0 }, want: "uid is required"},
		{name: "missing gid", mut: func(s *state.Sandbox) { s.GID = 0 }, want: "gid is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb := validSandbox()
			tt.mut(&sb)

			err := network.Teardown(cfg, sb)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("Teardown() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Teardown() = nil, want error")
			}
			if !strings.Contains(err.Error(), "network:") {
				t.Errorf("error %q missing network: prefix", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestSetupCommandSequence(t *testing.T) {
	cfg := testConfig(t)
	sb := validSandbox()

	var got []string
	restore := network.SetExecRunForTest(func(name string, args ...string) error {
		got = append(got, strings.Join(append([]string{name}, args...), " "))
		return nil
	})
	defer restore()

	if err := network.Setup(cfg, sb); err != nil {
		t.Fatalf("Setup() = %v", err)
	}

	want := []string{
		// Teardown first
		"ip route del 172.16.1.0/30",
		"ip link del kn1-host",
		"ip netns del kn-sandbox-0001",
		// create
		"ip netns add kn-sandbox-0001",
		"ip netns exec kn-sandbox-0001 ip link set lo up",
		"ip link add kn1-host type veth peer name kn1-ns",
		"ip link set kn1-ns netns kn-sandbox-0001",
		"ip addr add 10.200.1.1/30 dev kn1-host",
		"ip link set kn1-host up",
		"ip netns exec kn-sandbox-0001 ip addr add 10.200.1.2/30 dev kn1-ns",
		"ip netns exec kn-sandbox-0001 ip link set kn1-ns up",
		"ip netns exec kn-sandbox-0001 ip tuntap add dev tap0 mode tap user 12001 group 12001",
		"ip netns exec kn-sandbox-0001 ip addr add 172.16.1.1/30 dev tap0",
		"ip netns exec kn-sandbox-0001 ip link set tap0 up",
		"ip netns exec kn-sandbox-0001 sysctl -q -w net.ipv4.ip_forward=1",
		"ip netns exec kn-sandbox-0001 ip route replace default via 10.200.1.1 dev kn1-ns",
		"ip route replace 172.16.1.0/30 via 10.200.1.2 dev kn1-host",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d commands, want %d\ngot:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("command[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSetupCallsTeardownBeforeCreate(t *testing.T) {
	cfg := testConfig(t)
	order := []string{}

	restoreTeardown := network.SetTeardownFnForTest(func(*config.Config, state.Sandbox) error {
		order = append(order, "teardown")
		return nil
	})
	defer restoreTeardown()

	restoreRun := network.SetExecRunForTest(func(string, ...string) error {
		order = append(order, "create-step")
		return nil
	})
	defer restoreRun()

	if err := network.Setup(cfg, validSandbox()); err != nil {
		t.Fatalf("Setup() = %v", err)
	}
	if len(order) < 2 || order[0] != "teardown" {
		t.Fatalf("order = %v, want teardown first", order)
	}
	for _, s := range order[1:] {
		if s != "create-step" {
			t.Fatalf("order = %v, want only create-step after teardown", order)
		}
	}
}

func TestSetupPropagatesTeardownError(t *testing.T) {
	cfg := testConfig(t)
	want := errors.New("teardown blew up")
	restore := network.SetTeardownFnForTest(func(*config.Config, state.Sandbox) error {
		return want
	})
	defer restore()

	err := network.Setup(cfg, validSandbox())
	if !errors.Is(err, want) {
		t.Fatalf("Setup() = %v, want %v", err, want)
	}
}

func TestSetupBestEffortTeardownOnCreateFailure(t *testing.T) {
	cfg := testConfig(t)
	teardowns := 0
	restoreTeardown := network.SetTeardownFnForTest(func(*config.Config, state.Sandbox) error {
		teardowns++
		return nil
	})
	defer restoreTeardown()

	n := 0
	restoreRun := network.SetExecRunForTest(func(string, ...string) error {
		n++
		if n == 3 {
			return errors.New("boom")
		}
		return nil
	})
	defer restoreRun()

	err := network.Setup(cfg, validSandbox())
	if err == nil {
		t.Fatal("Setup() = nil, want error")
	}
	if teardowns != 2 {
		t.Fatalf("teardowns = %d, want 2 (before create + best-effort after failure)", teardowns)
	}
}

func TestSetupRefusesWhenRunning(t *testing.T) {
	cfg := testConfig(t)
	sb := validSandbox()
	pidFile := filepath.Join(cfg.Jailer.BaseDir, "firecracker", sb.ID, "root", "firecracker.pid")
	if err := os.MkdirAll(filepath.Dir(pidFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := network.Setup(cfg, sb)
	if !errors.Is(err, jailer.ErrRunning) {
		t.Fatalf("Setup() = %v, want jailer.ErrRunning", err)
	}
}

func TestTeardownRefusesWhenRunning(t *testing.T) {
	cfg := testConfig(t)
	sb := validSandbox()
	pidFile := filepath.Join(cfg.Jailer.BaseDir, "firecracker", sb.ID, "root", "firecracker.pid")
	if err := os.MkdirAll(filepath.Dir(pidFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := network.Teardown(cfg, sb)
	if !errors.Is(err, jailer.ErrRunning) {
		t.Fatalf("Teardown() = %v, want jailer.ErrRunning", err)
	}
}

func TestTeardownIgnoresMissingResources(t *testing.T) {
	cfg := testConfig(t)
	restore := network.SetExecRunForTest(func(string, ...string) error {
		return errors.New("not found")
	})
	defer restore()

	if err := network.Teardown(cfg, validSandbox()); err != nil {
		t.Fatalf("Teardown() = %v, want nil", err)
	}
}

func TestStatusNotImplemented(t *testing.T) {
	err := network.Status(validSandbox())
	if !errors.Is(err, network.ErrNotImplemented) {
		t.Fatalf("Status() = %v, want ErrNotImplemented", err)
	}
}

func TestEnsureHost(t *testing.T) {
	if err := network.EnsureHost(&config.Config{}); err != nil {
		t.Fatalf("EnsureHost() = %v, want nil", err)
	}
}
