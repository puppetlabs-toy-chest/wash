package errz

import "fmt"

// MatchError represents the case when the input tokens did not
// match a given node
type MatchError struct {
	reason string
}

func (m *MatchError) Error() string {
	return m.reason
}

// MatchErrorf creates a new MatchError object
func MatchErrorf(format string, a ...interface{}) error {
	return &MatchError{fmt.Sprintf(format, a...)}
}

// IsMatchError returns true if err is a MatchError,
// false otherwise.
func IsMatchError(err error) bool {
	_, ok := err.(*MatchError)
	return ok
}
