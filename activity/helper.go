package activity

import (
	"strconv"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"
)

// Cache pid to journals. This may end up getting the wrong process name if there are
// lots of new processes being created constantly, but makes fast things a *lot* faster.
var pidJournalCache = datastore.NewMemCache()

// JournalForPID creates a journal that can be used to record all wash-related activity induced
// by the given process ID. Journal ID is formatted as `<pid>-<name>-<createtime>`.
func JournalForPID(pid int) Journal {
	pidStr := strconv.Itoa(pid)
	result, err := pidJournalCache.GetOrUpdate("", pidStr, expires, true, func() (interface{}, error) {
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
			out.Start = time.Unix(created, 0)
		} else {
			log.Infof("Unable to get creation time for pid %v: %v", pid, err)
		}

		if cmdline, err := proc.Cmdline(); err == nil {
			out.Description = cmdline
		} else {
			log.Infof("Unable to get command-line for pid %v: %v", pid, err)
		}

		// We set Start based on the time we first encounter the process, not when the process was
		// started. This makes history make a little more sense when interacting with things like
		// the shell, which was likely started before wash was.
		out.Start = time.Now()
		return out, nil
	})

	if err != nil {
		log.Warnf("Unable to find pid %v: %v", pid, err)
		return Journal{ID: pidStr, Start: time.Now()}
	}

	return result.(Journal)
}
