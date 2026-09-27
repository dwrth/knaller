package network_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dwrth/knaller/internal/config"
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

func TestValidateRequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		mut   func(*state.Sandbox)
		want  string
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

			err := network.Teardown(sb)
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

func TestSetupCallsTeardownThenNotImplemented(t *testing.T) {
	called := false
	restore := network.SetTeardownFnForTest(func(state.Sandbox) error {
		called = true
		return nil
	})
	defer restore()

	err := network.Setup(validSandbox())
	if !called {
		t.Fatal("Setup did not call Teardown")
	}
	if !errors.Is(err, network.ErrNotImplemented) {
		t.Fatalf("Setup() = %v, want ErrNotImplemented", err)
	}
}

func TestSetupPropagatesTeardownError(t *testing.T) {
	want := errors.New("teardown blew up")
	restore := network.SetTeardownFnForTest(func(state.Sandbox) error {
		return want
	})
	defer restore()

	err := network.Setup(validSandbox())
	if !errors.Is(err, want) {
		t.Fatalf("Setup() = %v, want %v", err, want)
	}
}

func TestStatusNotImplemented(t *testing.T) {
	err := network.Status(validSandbox())
	if !errors.Is(err, network.ErrNotImplemented) {
		t.Fatalf("Status() = %v, want ErrNotImplemented", err)
	}
}

func TestStatusValidates(t *testing.T) {
	sb := validSandbox()
	sb.Namespace = ""
	err := network.Status(sb)
	if err == nil || !strings.Contains(err.Error(), "namespace is required") {
		t.Fatalf("Status() = %v, want namespace validation error", err)
	}
}

func TestEnsureHost(t *testing.T) {
	if err := network.EnsureHost(&config.Config{}); err != nil {
		t.Fatalf("EnsureHost() = %v, want nil", err)
	}
	if err := network.EnsureHost(nil); err != nil {
		t.Fatalf("EnsureHost(nil) = %v, want nil", err)
	}
}
