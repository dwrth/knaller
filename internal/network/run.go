package network

import (
	"fmt"
	"os/exec"
	"strings"
)

// execRun runs a command and returns combined output; tests may swap it.
var execRun = func(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	s := string(out)
	if err != nil {
		msg := strings.TrimSpace(s)
		if msg != "" {
			return s, fmt.Errorf("network: %s %s: %w: %s", name, strings.Join(args, " "), err, msg)
		}
		return s, fmt.Errorf("network: %s %s: %w", name, strings.Join(args, " "), err)
	}
	return s, nil
}

func run(name string, args ...string) error {
	_, err := execRun(name, args...)
	return err
}

func runOut(name string, args ...string) (string, error) {
	return execRun(name, args...)
}
