package lifecycle

import (
	"fmt"
	"os/exec"
	"strings"
)

// removeAccount deletes the Linux user and group for name.
// Absence is success (idempotent).
func removeAccount(name string) error {
	if name == "" {
		return fmt.Errorf("lifecycle: account name is required")
	}
	if err := runIgnoreMissing(exec.Command("userdel", name)); err != nil {
		return fmt.Errorf("lifecycle: userdel %s: %w", name, err)
	}
	if err := runIgnoreMissing(exec.Command("groupdel", name)); err != nil {
		return fmt.Errorf("lifecycle: groupdel %s: %w", name, err)
	}
	return nil
}

func runIgnoreMissing(cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	msg := strings.ToLower(string(out) + err.Error())
	if strings.Contains(msg, "does not exist") || strings.Contains(msg, "not found") {
		return nil
	}
	return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
}
