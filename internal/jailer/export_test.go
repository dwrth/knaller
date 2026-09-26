package jailer

import (
	"testing"
	"time"
)

// SetStopWaitForTest shortens Stop's poll window. Restored with t.Cleanup.
func SetStopWaitForTest(t *testing.T, wait, poll time.Duration) {
	t.Helper()
	oldWait, oldPoll := stopWait, stopPoll
	stopWait, stopPoll = wait, poll
	t.Cleanup(func() {
		stopWait, stopPoll = oldWait, oldPoll
	})
}

// StopInputsForTest runs stop against pre-built inputs (short socket paths on macOS).
func StopInputsForTest(in Inputs) error {
	return stop(in)
}
