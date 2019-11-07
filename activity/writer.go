package activity

import (
	"context"
	"strings"
)

// Writer logs the output as a call to Record per Write.
type Writer struct {
	context.Context
	Prefix string
}

func (a Writer) Write(p []byte) (int, error) {
	Record(a.Context, "%v: %v", a.Prefix, strings.TrimSpace(string(p)))
	return len(p), nil
}
