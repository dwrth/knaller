package network

import (
	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/jailer"
	"github.com/dwrth/knaller/internal/state"
)

func ensureStopped(cfg *config.Config, sandbox state.Sandbox) error {
	in, err := jailer.Resolve(cfg, sandbox)
	if err != nil {
		return err
	}
	return jailer.EnsureStopped(in)
}
