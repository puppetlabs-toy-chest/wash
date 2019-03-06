package plugin

import (
	"context"

	"github.com/puppetlabs/wash/journal"
	log "github.com/sirupsen/logrus"
)

// Record can be used to record plugin activity to a journal for later reference.
// It logs to a journal registered on the context via the Journal key. If no
// JournalID is registered, it instead sends to the server logs.
func Record(ctx context.Context, msg string, a ...interface{}) {
	obj := ctx.Value(Journal)
	if jid, ok := obj.(string); ok {
		if jid == "" {
			jid = "dead-letter-office"
		}
		journal.Record(jid, msg, a...)
	} else {
		log.Warnf(msg, a...)
	}
}
