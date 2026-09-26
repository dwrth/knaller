package jailer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dwrth/knaller/internal/config"
	"github.com/dwrth/knaller/internal/state"
)

// ErrStillRunning is returned when Stop's graceful shutdown did not finish before the wait deadline.
var ErrStillRunning = errors.New("jailer: sandbox did not stop before timeout")

var stopWait = 10 * time.Second
var stopPoll = 100 * time.Millisecond

// Stop requests a graceful guest shutdown via the Firecracker API socket, then
// waits for the jail process to exit. Missing API socket means already stopped.
// Stop does not remove jail files and does not mutate persisted sandbox state.
func Stop(cfg *config.Config, sandbox state.Sandbox) error {
	in, err := Resolve(cfg, sandbox)
	if err != nil {
		return err
	}
	return stop(in)
}

func stop(in Inputs) error {
	fi, err := os.Stat(in.APISock)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("jailer: stat api socket: %w", err)
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("jailer: api socket path is not a socket: %s", in.APISock)
	}

	// Match bash: best-effort request; process exit is the source of truth.
	_ = sendCtrlAltDel(in.APISock)

	return waitStopped(in)
}

func sendCtrlAltDel(sockPath string) error {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", sockPath)
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   2 * time.Second,
	}

	req, err := http.NewRequest(http.MethodPut, "http://localhost/actions", strings.NewReader(`{"action_type":"SendCtrlAltDel"}`))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("jailer: firecracker actions: status %d", resp.StatusCode)
	}
	return nil
}

func waitStopped(in Inputs) error {
	pid, ok, err := readPid(in.PidFile)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if !processAlive(pid) {
		return nil
	}

	deadline := time.Now().Add(stopWait)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return nil
		}
		time.Sleep(stopPoll)
	}
	if !processAlive(pid) {
		return nil
	}
	return fmt.Errorf("%w: pid %d", ErrStillRunning, pid)
}

func readPid(path string) (pid int, ok bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("jailer: read pidfile: %w", err)
	}
	pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false, nil
	}
	return pid, true, nil
}
