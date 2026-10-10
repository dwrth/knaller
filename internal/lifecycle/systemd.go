package lifecycle

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
)

var readyWait = 30 * time.Second
var readyPoll = 200 * time.Millisecond

// startSystemdUnit starts unit.
func startSystemdUnit(unit string) error {
	if unit == "" {
		return fmt.Errorf("lifecycle: systemd unit is required")
	}
	cmd := exec.Command("systemctl", "start", unit)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("lifecycle: systemctl start %s: %w: %s", unit, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// stopSystemdUnit stops unit. Missing/inactive units are success.
func stopSystemdUnit(unit string) error {
	if unit == "" {
		return fmt.Errorf("lifecycle: systemd unit is required")
	}
	cmd := exec.Command("systemctl", "stop", unit)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	msg := strings.ToLower(string(out) + err.Error())
	if strings.Contains(msg, "not loaded") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "inactive") {
		return nil
	}
	return fmt.Errorf("lifecycle: systemctl stop %s: %w: %s", unit, err, strings.TrimSpace(string(out)))
}

// waitSystemdActive polls until knaller@<id> is active or readyWait elapses.
func waitSystemdActive(_ *config.Config, sandbox state.Sandbox) error {
	unit := "knaller@" + sandbox.ID + ".service"
	deadline := time.Now().Add(readyWait)
	var last error
	for time.Now().Before(deadline) {
		cmd := exec.Command("systemctl", "is-active", "--quiet", unit)
		if err := cmd.Run(); err == nil {
			return nil
		} else {
			last = err
		}
		time.Sleep(readyPoll)
	}
	if last == nil {
		last = fmt.Errorf("not active")
	}
	return fmt.Errorf("lifecycle: %s not active before timeout: %w", unit, last)
}
