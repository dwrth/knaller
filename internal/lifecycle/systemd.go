package lifecycle

import (
	"fmt"
	"os/exec"
	"strings"
)

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
	// systemctl stop on a never-started unit often still exits 0; if the
	// binary is missing in tests/dev hosts, treat as non-fatal only when
	// callers inject stopUnit. Real hosts must have systemctl.
	return fmt.Errorf("lifecycle: systemctl stop %s: %w: %s", unit, err, strings.TrimSpace(string(out)))
}
