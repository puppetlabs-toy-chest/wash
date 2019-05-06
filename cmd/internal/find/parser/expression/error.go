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