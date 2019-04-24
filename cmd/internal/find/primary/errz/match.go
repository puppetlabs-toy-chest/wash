package errz

// MatchError represents the case when the input tokens did not
// match the given parser.
type MatchError struct {
	reason string
}

func (m MatchError) Error() string {
	return m.reason
}

// NewMatchError creates a new MatchError object
func NewMatchError(reason string) MatchError {
	return MatchError{reason}
}

// IsMatchError returns true if err is a MatchError,
// false otherwise.
func IsMatchError(err error) bool {
	_, ok := err.(MatchError)
	return ok
}
