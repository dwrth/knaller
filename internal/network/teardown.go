package network

import (
	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
)

// teardownFn is the real Teardown body; tests may swap it to observe call order.
var teardownFn = teardown

// Teardown removes per-sandbox network resources.
// Missing resources are not an error (idempotent).
// Refuses if the sandbox Firecracker process is still alive.
func Teardown(cfg *config.Config, sandbox state.Sandbox) error {
	return teardownFn(cfg, sandbox)
}

func teardown(cfg *config.Config, sandbox state.Sandbox) error {
	if err := validate(sandbox); err != nil {
		return err
	}
	if err := ensureStopped(cfg, sandbox); err != nil {
		return err
	}

	// Best-effort deletes; absence is success.
	_ = run("ip", "route", "del", sandbox.GuestSubnet)
	_ = run("ip", "link", "del", sandbox.HostVeth)
	_ = run("ip", "netns", "del", sandbox.Namespace)
	return nil
}
