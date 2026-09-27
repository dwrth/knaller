package network

import (
	"testing"
)

func TestAddrWithPrefix(t *testing.T) {
	got, err := addrWithPrefix("10.200.1.1", "10.200.1.0/30")
	if err != nil {
		t.Fatal(err)
	}
	if got != "10.200.1.1/30" {
		t.Fatalf("got %q, want 10.200.1.1/30", got)
	}

	if _, err := addrWithPrefix("10.200.2.1", "10.200.1.0/30"); err == nil {
		t.Fatal("expected error for address outside subnet")
	}
}
