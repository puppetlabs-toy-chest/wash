package cmdfind

import (
	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
)

// Entry represents an entry as interpreted by `wash find`.
// It is primarily needed for its NormalizedPath field
type Entry struct {
	*apitypes.Entry
	NormalizedPath string
}

func newEntry() Entry {
	return Entry{Entry: &apitypes.Entry{}}
}

// Info is a wrapper to c.Info
func Info(c *client.DomainSocketClient, path string) (Entry, error) {
	e, err := c.Info(path)
	if err != nil {
		return Entry{}, err
	}
	return Entry{
		Entry:          &e,
		NormalizedPath: path,
	}, nil
}

// List is a wrapper to c.List that handles normalizing the children's
// path relative to e's normalized path
func List(c *client.DomainSocketClient, e Entry) ([]Entry, error) {
	rawChildren, err := c.List(e.Path)
	if err != nil {
		return nil, err
	}
	children := make([]Entry, len(rawChildren))
	for i, ch := range rawChildren {
		normalizedPath := e.NormalizedPath
		normalizedPath += "/"
		normalizedPath += ch.CName

		children[i] = Entry{
			Entry:          &ch,
			NormalizedPath: normalizedPath,
		}
	}
	return children, nil
}
