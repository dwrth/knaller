package network

import (
	"strconv"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
)

// Setup creates per-sandbox network resources (netns, veth, TAP, routes).
// Contract: refuse if running, Teardown first, then create.
// On create failure, Teardown is attempted again best-effort.
func Setup(cfg *config.Config, sandbox state.Sandbox) error {
	if err := validate(sandbox); err != nil {
		return err
	}
	if err := ensureStopped(cfg, sandbox); err != nil {
		return err
	}
	if err := Teardown(cfg, sandbox); err != nil {
		return err
	}
	if err := create(sandbox); err != nil {
		_ = Teardown(cfg, sandbox)
		return err
	}
	return nil
}

func create(sandbox state.Sandbox) error {
	hostAddr, err := addrWithPrefix(sandbox.TransitHostIP, sandbox.TransitSubnet)
	if err != nil {
		return err
	}
	nsAddr, err := addrWithPrefix(sandbox.TransitNSIP, sandbox.TransitSubnet)
	if err != nil {
		return err
	}
	tapAddr, err := addrWithPrefix(sandbox.GatewayIP, sandbox.GuestSubnet)
	if err != nil {
		return err
	}

	ns := sandbox.Namespace
	uid := strconv.Itoa(sandbox.UID)
	gid := strconv.Itoa(sandbox.GID)

	steps := [][]string{
		{"ip", "netns", "add", ns},
		{"ip", "netns", "exec", ns, "ip", "link", "set", "lo", "up"},
		{"ip", "link", "add", sandbox.HostVeth, "type", "veth", "peer", "name", sandbox.NSVeth},
		{"ip", "link", "set", sandbox.NSVeth, "netns", ns},
		{"ip", "addr", "add", hostAddr, "dev", sandbox.HostVeth},
		{"ip", "link", "set", sandbox.HostVeth, "up"},
		{"ip", "netns", "exec", ns, "ip", "addr", "add", nsAddr, "dev", sandbox.NSVeth},
		{"ip", "netns", "exec", ns, "ip", "link", "set", sandbox.NSVeth, "up"},
		{"ip", "netns", "exec", ns, "ip", "tuntap", "add", "dev", sandbox.TAP, "mode", "tap", "user", uid, "group", gid},
		{"ip", "netns", "exec", ns, "ip", "addr", "add", tapAddr, "dev", sandbox.TAP},
		{"ip", "netns", "exec", ns, "ip", "link", "set", sandbox.TAP, "up"},
		{"ip", "netns", "exec", ns, "sysctl", "-q", "-w", "net.ipv4.ip_forward=1"},
		{"ip", "netns", "exec", ns, "ip", "route", "replace", "default", "via", sandbox.TransitHostIP, "dev", sandbox.NSVeth},
		{"ip", "route", "replace", sandbox.GuestSubnet, "via", sandbox.TransitNSIP, "dev", sandbox.HostVeth},
	}
	for _, args := range steps {
		if err := run(args[0], args[1:]...); err != nil {
			return err
		}
	}
	return nil
}
