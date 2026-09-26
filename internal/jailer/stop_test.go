package jailer_test

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/dwrth/knaller/internal/jailer"
)

func TestStopAlreadyStopped(t *testing.T) {
	cfg, sb := prepareEnv(t)
	if err := jailer.Stop(cfg, sb); err != nil {
		t.Fatal(err)
	}
}

func shortStopInputs(t *testing.T) jailer.Inputs {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "kn")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return jailer.Inputs{
		ID:      "test",
		APISock: filepath.Join(dir, "fc.sock"),
		PidFile: filepath.Join(dir, "fc.pid"),
	}
}

func TestStopGraceful(t *testing.T) {
	in := shortStopInputs(t)

	ln, err := net.Listen("unix", in.APISock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	gotAction := make(chan string, 1)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/actions" {
			http.NotFound(w, r)
			return
		}
		var body struct {
			ActionType string `json:"action_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		gotAction <- body.ActionType
		w.WriteHeader(http.StatusNoContent)
	})}
	go srv.Serve(ln)
	defer srv.Close()

	if err := os.WriteFile(in.PidFile, []byte("2147483647"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := jailer.StopInputsForTest(in); err != nil {
		t.Fatal(err)
	}

	select {
	case action := <-gotAction:
		if action != "SendCtrlAltDel" {
			t.Fatalf("action = %q", action)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("API not called")
	}
}

func TestStopTimeout(t *testing.T) {
	in := shortStopInputs(t)

	ln, err := net.Listen("unix", in.APISock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	if err := os.WriteFile(in.PidFile, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}

	jailer.SetStopWaitForTest(t, 200*time.Millisecond, 50*time.Millisecond)

	err = jailer.StopInputsForTest(in)
	if !errors.Is(err, jailer.ErrStillRunning) {
		t.Fatalf("error = %v, want ErrStillRunning", err)
	}
}
