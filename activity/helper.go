package activity

import (
	"strconv"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"
)

// Cache pid to process names. This may end up getting the wrong process name if there are
// lots of new processes being created constantly, but makes fast things a *lot* faster.
var pidToExec = datastore.NewMemCache()

// JournalForPID converts a process ID to a journal ID, and the command invocation of that
// process ID. The journal ID includes executable name to make it easier to identify, and creation
// timestamp to make it unique.
func JournalForPID(pid int) Journal {
	pidStr := strconv.Itoa(pid)
	result, err := pidToExec.GetOrUpdate("", pidStr, expires, true, func() (interface{}, error) {
		proc, err := process.NewProcess(int32(pid))
		var out Journal
		if err != nil {
			return out, err
		}

		out.ID = pidStr
		if name, err := proc.Name(); err == nil && name != "" {
			out.ID += "-" + name
		} else {
			log.Infof("Unable to get name for pid %v: %v", pid, err)
		}

		if created, err := proc.CreateTime(); err == nil {
			out.ID += "-" + strconv.FormatInt(created, 10)
		} else {
			log.Infof("Unable to get creation time for pid %v: %v", pid, err)
		}

		if cmdline, err := proc.Cmdline(); err == nil {
			out.Description = cmdline
		} else {
			log.Infof("Unable to get command-line for pid %v: %v", pid, err)
		}
		out.Start = time.Now()
		return out, nil
	})

	if err != nil {
		log.Warnf("Unable to find pid %v: %v", pid, err)
		return Journal{ID: pidStr}
	}

	return result.(Journal)
}
