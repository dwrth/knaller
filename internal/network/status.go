package network

import (
	"strings"

	"github.com/dwrth/knaller/internal/state"
)

// Report is the observed presence of per-sandbox dataplane resources on the host.
type Report struct {
	NetNS      bool
	HostVeth   bool
	GuestRoute bool
}

// Present reports whether any tracked resource still exists.
func (r Report) Present() bool {
	return r.NetNS || r.HostVeth || r.GuestRoute
}

// Status probes netns, host veth, and guest route for sandbox.
func Status(sandbox state.Sandbox) (Report, error) {
	if err := validate(sandbox); err != nil {
		return Report{}, err
	}

	var r Report
	r.NetNS = run("ip", "netns", "exec", sandbox.Namespace, "true") == nil
	r.HostVeth = run("ip", "link", "show", sandbox.HostVeth) == nil

	out, err := runOut("ip", "route", "show", sandbox.GuestSubnet)
	r.GuestRoute = err == nil && strings.TrimSpace(out) != ""

	return r, nil
}
