package main

import (
	"testing"
)

func TestCreateRequiresFlags(t *testing.T) {
	cfg := writeTestConfig(t, t.TempDir())
	if code := run([]string{"create", "--config", cfg, "--name", "web-1"}); code != 2 {
		t.Fatalf("create missing flags = %d, want 2", code)
	}
}

func TestDeleteMissing(t *testing.T) {
	cfg := writeTestConfig(t, t.TempDir())
	if code := run([]string{"delete", "--config", cfg, "nope"}); code != 1 {
		t.Fatalf("delete missing = %d, want 1", code)
	}
}

func TestDeleteRequiresRef(t *testing.T) {
	cfg := writeTestConfig(t, t.TempDir())
	if code := run([]string{"delete", "--config", cfg}); code != 2 {
		t.Fatalf("delete without ref = %d, want 2", code)
	}
}
