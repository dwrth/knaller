package network

import (
	"fmt"
	"os/exec"
	"strings"
)

// execRun runs a command; tests may swap it to record argv without touching Linux.
var execRun = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("network: %s %s: %w: %s", name, strings.Join(args, " "), err, msg)
		}
		return fmt.Errorf("network: %s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func run(name string, args ...string) error {
	return execRun(name, args...)
}
