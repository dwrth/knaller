package jailer

import (
	"fmt"
	"os"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
)

// Remove deletes the disposable jail directory for sandbox.
// It refuses if a live Firecracker process is recorded in the jail pidfile.
// A missing jail directory is success (idempotent).
func Remove(cfg *config.Config, sandbox state.Sandbox) error {
	in, err := Resolve(cfg, sandbox)
	if err != nil {
		return err
	}
	if err := EnsureStopped(in); err != nil {
		return err
	}
	if err := os.RemoveAll(in.JailDir); err != nil {
		return fmt.Errorf("jailer: remove jail: %w", err)
	}
	return nil
}
