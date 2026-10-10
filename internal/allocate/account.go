package allocate

import "fmt"

// AccountName returns the Linux user/group name for slot.
func AccountName(slot int) string {
	return fmt.Sprintf("kn-%04d", slot)
}
