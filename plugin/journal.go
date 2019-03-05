package plugin

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type journaler interface {
	Log(string)
}

// Log can be used to log plugin activity to a journal for later reference.
// It logs to a journal registered on the context via the Journal key.
func Log(ctx context.Context, msg string) {
	obj := ctx.Value(Journal)
	if jnl, ok := obj.(journaler); ok {
		jnl.Log(msg)
	} else {
		log.Warnf("Unable to log to journal: %v", msg)
	}
}
