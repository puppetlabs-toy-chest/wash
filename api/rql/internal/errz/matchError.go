package errz

import "fmt"

// MatchError represents the case when the input tokens did not
// match a given node
type MatchError struct {
	reason string
}

func (m MatchError) Error() string {
	return m.reason
}

// MatchErrorf creates a new MatchError object
func MatchErrorf(format string, a ...interface{}) MatchError {
	return MatchError{fmt.Sprintf(format, a...)}
}

// IsMatchError returns true if err is a MatchError,
// false otherwise.
func IsMatchError(err error) bool {
	_, ok := err.(MatchError)
	return ok
}

// IsSyntaxError returns true if err is a syntax error, false otherwise.
func IsSyntaxError(err error) bool {
	if err == nil {
		return false
	}
	switch err.(type) {
	case MatchError:
		return false
	default:
		return true
	}
}
