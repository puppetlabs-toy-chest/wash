package journal

import (
	"strconv"

	"github.com/mitchellh/go-ps"
)

// PIDToID converts a process ID to a journal ID, appending the name of the executable
// running that process if we can retrieve it.
func PIDToID(pid int) string {
	journalid := strconv.FormatInt(int64(pid), 10)
	// Include the executable name if we can find it.
	proc, err := ps.FindProcess(pid)
	if err == nil && proc != nil {
		journalid += "-" + proc.Executable()
	}
	return journalid
}
