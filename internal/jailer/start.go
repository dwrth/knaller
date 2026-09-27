package jailer

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
)

const (
	jailerPath      = "/usr/local/bin/jailer"
	firecrackerPath = "/usr/local/bin/firecracker"
)

// Start resolves sandbox inputs, checks the jail is ready, then replaces the
// current process with jailer (like bash exec). It does not call Prepare and
// does not mutate persisted sandbox state.
func Start(cfg *config.Config, sandbox state.Sandbox) error {
	in, err := Resolve(cfg, sandbox)
	if err != nil {
		return err
	}
	if err := ready(in); err != nil {
		return err
	}
	argv := Argv(in)
	return syscall.Exec(argv[0], argv, os.Environ())
}

// Argv returns the jailer argument vector (including argv0) for in.
func Argv(in Inputs) []string {
	return []string{
		jailerPath,
		"--id", in.ID,
		"--exec-file", firecrackerPath,
		"--uid", strconv.Itoa(in.UID),
		"--gid", strconv.Itoa(in.GID),
		"--chroot-base-dir", in.ChrootBaseDir,
		"--netns", in.NetNSPath,
		"--cgroup-version", "2",
		"--resource-limit", "no-file=1024",
		"--",
		"--api-sock", "/run/firecracker.socket",
		"--config-file", "/config.json",
	}
}

func ready(in Inputs) error {
	if err := EnsureStopped(in); err != nil {
		return err
	}
	if _, err := os.Stat(in.NetNSPath); err != nil {
		return fmt.Errorf("jailer: network namespace: %w", err)
	}
	for _, name := range []string{"vmlinux", "rootfs.ext4", "config.json"} {
		path := filepath.Join(in.RootDir, name)
		fi, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("jailer: jail missing %s: %w", path, err)
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("jailer: jail entry %s is not a regular file", path)
		}
	}
	return nil
}
