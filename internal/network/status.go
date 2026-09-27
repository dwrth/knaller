package network

import (
	"errors"

	"github.com/dwrth/knaller/internal/state"
)

// ErrNotImplemented is returned when Status observation is not wired yet.
var ErrNotImplemented = errors.New("network: not implemented")

// Status reports observed network resources for sandbox.
func Status(sandbox state.Sandbox) error {
	if err := validate(sandbox); err != nil {
		return err
	}
	return ErrNotImplemented
}
