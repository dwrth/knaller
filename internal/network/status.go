package network

import "github.com/dwrth/knaller/internal/state"

// Status reports observed network resources for sandbox.
// Observation is not implemented until dataplane Setup exists.
func Status(sandbox state.Sandbox) error {
	if err := validate(sandbox); err != nil {
		return err
	}
	return ErrNotImplemented
}
