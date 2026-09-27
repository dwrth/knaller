package network

import "github.com/dwrth/knaller/internal/state"

// SetTeardownFnForTest swaps Teardown's implementation for call-order tests.
// The returned function restores the previous implementation.
func SetTeardownFnForTest(fn func(state.Sandbox) error) func() {
	prev := teardownFn
	teardownFn = fn
	return func() { teardownFn = prev }
}
