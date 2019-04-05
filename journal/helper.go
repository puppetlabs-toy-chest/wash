package journal

import (
	"strconv"

	"github.com/mitchellh/go-ps"
	"github.com/puppetlabs/wash/datastore"
)

// Cache pid to process names. This may end up getting the wrong process name if there are
// lots of new processes being created constantly, but makes fast things a *lot* faster.
var pidToExec = datastore.NewMemCache()

// PIDToID converts a process ID to a journal ID, appending the name of the executable
// running that process if we can retrieve it.
func PIDToID(pid int) string {
	pidStr := strconv.FormatInt(int64(pid), 10)
	result, err := pidToExec.GetOrUpdate("", pidStr, expires, true, func() (interface{}, error) {
		journalid := pidStr
		// Include the executable name if we can find it.
		proc, err := ps.FindProcess(pid)
		if err == nil && proc != nil {
			journalid += "-" + proc.Executable()
		}
		return journalid, nil
	})

	if err != nil {
		return pidStr
	}
	return result.(string)
}
