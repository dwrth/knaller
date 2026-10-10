package allocate_test

import (
	"testing"

	"github.com/dwrth/knaller/internal/allocate"
)

func TestAccountName(t *testing.T) {
	if got := allocate.AccountName(4); got != "kn-0004" {
		t.Fatalf("AccountName(4) = %q, want kn-0004", got)
	}
}
