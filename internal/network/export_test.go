package network

import (
	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
)

// SetTeardownFnForTest swaps Teardown's implementation for call-order tests.
func SetTeardownFnForTest(fn func(*config.Config, state.Sandbox) error) func() {
	prev := teardownFn
	teardownFn = fn
	return func() { teardownFn = prev }
}

// SetExecRunForTest swaps the command runner. Returns a restore function.
func SetExecRunForTest(fn func(name string, args ...string) error) func() {
	prev := execRun
	execRun = fn
	return func() { execRun = prev }
}
