package apitypes

// JournalID is the query key used to provide a journal ID to assocate multiple actions.
// If not provided in a request, any journal entries will be sent to the dead letter office.
const JournalID = "journal"
