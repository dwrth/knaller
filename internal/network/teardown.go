package network

import "github.com/dwrth/knaller/internal/state"

// teardownFn is the real Teardown body; tests may swap it to observe call order.
var teardownFn = teardown

// Teardown removes per-sandbox network resources.
// Missing resources are not an error (idempotent).
func Teardown(sandbox state.Sandbox) error {
	return teardownFn(sandbox)
}

func teardown(sandbox state.Sandbox) error {
	if err := validate(sandbox); err != nil {
		return err
	}
	// 12.2+: delete route, veth, netns.
	return nil
}
