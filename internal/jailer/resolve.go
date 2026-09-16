// Package jailer resolves and (later) manages Firecracker jailer supervision inputs.
package jailer

import (
	"fmt"
	"path/filepath"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
	"github.com/dwrth/knaller/internal/storage"
)

// Inputs are the concrete values prepare/start/stop need for one sandbox.
type Inputs struct {
	ID string

	UID int
	GID int

	NetNSPath string

	ChrootBaseDir string
	JailDir       string
	RootDir       string

	SandboxDir string
	Kernel     string
	Rootfs     string
	Config     string

	APISock string
	PidFile string
}

// Resolve builds jailer inputs from config and persisted sandbox state.
// It does not check that durable artifacts or the jail exist on disk.
func Resolve(cfg *config.Config, sandbox state.Sandbox) (Inputs, error) {
	if sandbox.ID == "" {
		return Inputs{}, fmt.Errorf("jailer: sandbox id is required")
	}
	if sandbox.Namespace == "" {
		return Inputs{}, fmt.Errorf("jailer: sandbox %s: namespace is required", sandbox.ID)
	}
	if cfg.Jailer.BaseDir == "" {
		return Inputs{}, fmt.Errorf("jailer: jailer.base_dir is required")
	}
	if cfg.Firecracker.SandboxDirectory == "" {
		return Inputs{}, fmt.Errorf("jailer: firecracker.sandbox_directory is required")
	}

	m := storage.New(cfg)
	jailDir := filepath.Join(cfg.Jailer.BaseDir, "firecracker", sandbox.ID)
	rootDir := filepath.Join(jailDir, "root")

	return Inputs{
		ID:  sandbox.ID,
		UID: sandbox.UID,
		GID: sandbox.GID,

		NetNSPath: filepath.Join("/run/netns", sandbox.Namespace),

		ChrootBaseDir: cfg.Jailer.BaseDir,
		JailDir:       jailDir,
		RootDir:       rootDir,

		SandboxDir: m.SandboxDir(sandbox.ID),
		Kernel:     m.Kernel(sandbox.ID),
		Rootfs:     m.Rootfs(sandbox.ID),
		Config:     m.Config(sandbox.ID),

		APISock: filepath.Join(rootDir, "run", "firecracker.socket"),
		PidFile: filepath.Join(rootDir, "firecracker.pid"),
	}, nil
}

// ResolveID loads sandbox state by id, then resolves jailer inputs.
func ResolveID(cfg *config.Config, store *state.Store, id string) (Inputs, error) {
	sandbox, err := store.Load(id)
	if err != nil {
		return Inputs{}, err
	}
	return Resolve(cfg, sandbox)
}
