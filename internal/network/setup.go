package network

import (
	"errors"

	"github.com/dwrth/knaller/internal/state"
)

// ErrNotImplemented is returned when a Stage 12.1 stub has no dataplane yet.
var ErrNotImplemented = errors.New("network: not implemented")

// Setup creates per-sandbox network resources (netns, veth, TAP, routes).
// Contract: Teardown first, then create. Create is not implemented until 12.2.
func Setup(sandbox state.Sandbox) error {
	if err := validate(sandbox); err != nil {
		return err
	}
	if err := Teardown(sandbox); err != nil {
		return err
	}
	return ErrNotImplemented
}
