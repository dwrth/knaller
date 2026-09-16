package jailer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
)

// ErrRunning is returned when Prepare would destroy a jail that still has a live process.
var ErrRunning = errors.New("jailer: sandbox appears to be running")

// Prepare rebuilds the disposable jail chroot for sandbox from durable artifacts.
// It refuses if a live Firecracker process is recorded in the jail pidfile.
// On failure after the jail directory was removed, the jail directory is removed again.
func Prepare(cfg *config.Config, sandbox state.Sandbox) error {
	in, err := Resolve(cfg, sandbox)
	if err != nil {
		return err
	}
	if err := ensureStopped(in); err != nil {
		return err
	}
	if err := ensureArtifacts(in); err != nil {
		return err
	}

	if err := os.RemoveAll(in.JailDir); err != nil {
		return fmt.Errorf("jailer: remove jail: %w", err)
	}

	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(in.JailDir)
		}
	}()

	if err := os.MkdirAll(in.RootDir, 0o700); err != nil {
		return fmt.Errorf("jailer: mkdir jail: %w", err)
	}

	links := []struct{ src, dst string }{
		{in.Kernel, filepath.Join(in.RootDir, "vmlinux")},
		{in.Rootfs, filepath.Join(in.RootDir, "rootfs.ext4")},
		{in.Config, filepath.Join(in.RootDir, "config.json")},
	}
	for _, l := range links {
		if err := os.Link(l.src, l.dst); err != nil {
			return fmt.Errorf("jailer: link %s: %w", l.dst, err)
		}
	}

	ok = true
	return nil
}

func ensureStopped(in Inputs) error {
	data, err := os.ReadFile(in.PidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("jailer: read pidfile: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return nil
	}
	if processAlive(pid) {
		return fmt.Errorf("%w: pid %d", ErrRunning, pid)
	}
	return nil
}

func processAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func ensureArtifacts(in Inputs) error {
	for _, path := range []string{in.Kernel, in.Rootfs, in.Config} {
		fi, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("jailer: missing artifact %s: %w", path, err)
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("jailer: artifact %s is not a regular file", path)
		}
	}
	return nil
}
