package lifecycle

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ensureAccount creates a system group/user for name with uid/gid, or verifies
// an existing account matches. Mirrors scripts/install-host.sh ensure_account.
func ensureAccount(name string, uid, gid int) error {
	if name == "" {
		return fmt.Errorf("lifecycle: account name is required")
	}
	if uid <= 0 || gid <= 0 {
		return fmt.Errorf("lifecycle: uid/gid must be positive")
	}

	if err := ensureGroup(name, gid); err != nil {
		return err
	}
	return ensureUser(name, uid, gid)
}

func ensureGroup(name string, gid int) error {
	out, err := exec.Command("getent", "group", name).CombinedOutput()
	if err == nil {
		current, parseErr := parseGetentGID(string(out))
		if parseErr != nil {
			return parseErr
		}
		if current != gid {
			return fmt.Errorf("lifecycle: group %s exists with gid %d, want %d", name, current, gid)
		}
		return nil
	}
	if exec.Command("getent", "group", strconv.Itoa(gid)).Run() == nil {
		return fmt.Errorf("lifecycle: gid %d already belongs to another group", gid)
	}
	cmd := exec.Command("groupadd", "--system", "--gid", strconv.Itoa(gid), name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("lifecycle: groupadd %s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func ensureUser(name string, uid, gid int) error {
	out, err := exec.Command("getent", "passwd", name).CombinedOutput()
	if err == nil {
		current, parseErr := parseGetentUID(string(out))
		if parseErr != nil {
			return parseErr
		}
		if current != uid {
			return fmt.Errorf("lifecycle: user %s exists with uid %d, want %d", name, current, uid)
		}
		return nil
	}
	if exec.Command("getent", "passwd", strconv.Itoa(uid)).Run() == nil {
		return fmt.Errorf("lifecycle: uid %d already belongs to another user", uid)
	}
	cmd := exec.Command(
		"useradd",
		"--system",
		"--uid", strconv.Itoa(uid),
		"--gid", strconv.Itoa(gid),
		"--no-create-home",
		"--shell", "/usr/sbin/nologin",
		name,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("lifecycle: useradd %s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func parseGetentGID(line string) (int, error) {
	// name:x:GID:members
	parts := strings.Split(strings.TrimSpace(line), ":")
	if len(parts) < 3 {
		return 0, fmt.Errorf("lifecycle: parse group getent: %q", line)
	}
	return strconv.Atoi(parts[2])
}

func parseGetentUID(line string) (int, error) {
	// name:x:UID:GID:...
	parts := strings.Split(strings.TrimSpace(line), ":")
	if len(parts) < 3 {
		return 0, fmt.Errorf("lifecycle: parse passwd getent: %q", line)
	}
	return strconv.Atoi(parts[2])
}

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
