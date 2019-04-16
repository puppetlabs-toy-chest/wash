package activity

import (
	"strconv"

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
		out := NewJournal(pidStr, "")
		if err != nil {
			return out, err
		}

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
		return out, nil
	})

	if err != nil {
		log.Warnf("Unable to find pid %v: %v", pid, err)
		return NewJournal(pidStr, "")
	}

	return result.(Journal)
}
