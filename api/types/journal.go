package apitypes

import "time"

// JournalIDHeader is the name of the HTTP Header used to provide a journal ID to assocate multiple actions.
const JournalIDHeader = "JournalID"

// JournalDescHeader is the name of the HTTP Header used to provide a more detailed description of the activity
// related to that journal entry, to be displayed as part of the history.
const JournalDescHeader = "JournalDesc"

// Activity describes an activity from wash's `activity.History`.
type Activity struct {
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
}

// HistoryResponse describes the result returned by the `/history` endpoint.
//
// swagger:response
type HistoryResponse struct {
	// in: body
	Activities []Activity
}
