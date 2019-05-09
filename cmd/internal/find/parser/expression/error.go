package expression

import (
	"fmt"
)

// UnknownTokenError represents an unknown token that was found
// when parsing the expression
type UnknownTokenError struct {
	Token string
}

func (e UnknownTokenError) Error() string {
	return fmt.Sprintf("unknown token %v", e.Token)
}

// EmptyExpressionError represents an empty expression error.
type EmptyExpressionError struct {
	msg string
}

// NewEmptyExpressionError creates an EmptyExpressionError object
func NewEmptyExpressionError(msg string) EmptyExpressionError {
	return EmptyExpressionError{msg}
}

func (e EmptyExpressionError) Error() string {
	return e.msg
}

// IsEmptyExpressionError returns true if err is an EmptyExpressionError
// object, false otherwise.
func IsEmptyExpressionError(err error) bool {
	_, ok := err.(EmptyExpressionError)
	return ok
}