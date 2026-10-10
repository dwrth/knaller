// Package lifecycle orchestrates sandbox create and destroy with rollback.
package lifecycle

import (
	"errors"
	"path/filepath"

	"github.com/dwrth/knaller/internal/capacity"
	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/jailer"
	"github.com/dwrth/knaller/internal/lock"
	"github.com/dwrth/knaller/internal/network"
	"github.com/dwrth/knaller/internal/state"
	"github.com/dwrth/knaller/internal/storage"
)

// ErrBusy is returned when the node mutation lock is already held.
var ErrBusy = lock.ErrBusy

// ErrNameTaken is returned when a sandbox name is already in use.
var ErrNameTaken = errors.New("lifecycle: sandbox name already taken")

// Manager coordinates node-local sandbox lifecycle mutations.
type Manager struct {
	cfg     *config.Config
	store   *state.Store
	lock    *lock.Lock
	storage *storage.Manager

	collect          func(*config.Config) (capacity.Capacity, error)
	ensureAcct       func(name string, uid, gid int) error
	ensureHost       func(*config.Config) error
	provisionStorage func(state.Sandbox) error
	startUnit        func(unit string) error
	waitReady        func(*config.Config, state.Sandbox) error
	stopRuntime      func(*config.Config, state.Sandbox) error
	stopUnit         func(unit string) error
	teardownNet      func(*config.Config, state.Sandbox) error
	removeJail       func(*config.Config, state.Sandbox) error
	removeAcct       func(name string) error
}

// New returns a Manager using cfg and a lock file under the state directory.
func New(cfg *config.Config) *Manager {
	store := storage.New(cfg)
	return &Manager{
		cfg:              cfg,
		store:            state.New(cfg.State.Directory),
		lock:             lock.New(filepath.Join(cfg.State.Directory, "knaller.lock")),
		storage:          store,
		collect:          capacity.Collect,
		ensureAcct:       ensureAccount,
		ensureHost:       network.EnsureHost,
		provisionStorage: defaultProvisionStorage(store),
		startUnit:        startSystemdUnit,
		waitReady:        waitSystemdActive,
		stopRuntime:      jailer.Stop,
		stopUnit:         stopSystemdUnit,
		teardownNet:      network.Teardown,
		removeJail:       jailer.Remove,
		removeAcct:       removeAccount,
	}
}
