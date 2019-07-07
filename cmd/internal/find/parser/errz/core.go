package errz

// MatchError represents the case when the input tokens did not
// match a given parser.
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

// UnknownTokenError represents an unknown token that was found
// when parsing the expression
type UnknownTokenError struct {
	Token string
	Msg   string
}

func (e UnknownTokenError) Error() string {
	return e.Msg
}

// IsSyntaxError returns true if err is a syntax error, false otherwise.
func IsSyntaxError(err error) bool {
	if err == nil {
		return false
	}
	switch err.(type) {
	case MatchError:
		return false
	case UnknownTokenError:
		return false
	default:
		return true
	}
}
