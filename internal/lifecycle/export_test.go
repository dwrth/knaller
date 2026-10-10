package lifecycle

import (
	"github.com/dwrth/knaller/internal/capacity"
	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
)

func (m *Manager) SetCollectForTest(fn func(*config.Config) (capacity.Capacity, error)) {
	m.collect = fn
}

func (m *Manager) SetEnsureAcctForTest(fn func(name string, uid, gid int) error) {
	m.ensureAcct = fn
}

func (m *Manager) SetEnsureHostForTest(fn func(*config.Config) error) {
	m.ensureHost = fn
}

func (m *Manager) SetProvisionStorageForTest(fn func(state.Sandbox) error) {
	m.provisionStorage = fn
}

func (m *Manager) SetStartUnitForTest(fn func(unit string) error) {
	m.startUnit = fn
}

func (m *Manager) SetWaitReadyForTest(fn func(*config.Config, state.Sandbox) error) {
	m.waitReady = fn
}

func (m *Manager) SetStopRuntimeForTest(fn func(*config.Config, state.Sandbox) error) {
	m.stopRuntime = fn
}

func (m *Manager) SetStopUnitForTest(fn func(unit string) error) {
	m.stopUnit = fn
}

func (m *Manager) SetTeardownNetForTest(fn func(*config.Config, state.Sandbox) error) {
	m.teardownNet = fn
}

func (m *Manager) SetRemoveJailForTest(fn func(*config.Config, state.Sandbox) error) {
	m.removeJail = fn
}

func (m *Manager) SetRemoveAcctForTest(fn func(name string) error) {
	m.removeAcct = fn
}
