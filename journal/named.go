package journal

// NamedJournal represents a journal that knows the ID to write to.
type NamedJournal struct {
	ID string
}

// Log writes to the named journal.
func (n NamedJournal) Log(msg string, a ...interface{}) {
	id := n.ID
	if n.ID == "" {
		id = "dead-letter-office"
	}
	Log(id, msg, a...)
}
