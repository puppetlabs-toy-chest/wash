package cmdfind

import (
	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
)

type entry struct {
	apitypes.Entry
	NormalizedPath string
}

// info is a wrapper to c.Info
func info(c *client.DomainSocketClient, path string) (entry, error) {
	e, err := c.Info(path)
	if err != nil {
		return entry{}, err
	}
	return entry{
		Entry:          e,
		NormalizedPath: path,
	}, nil
}

// list is a wrapper to c.List that handles normalizing the children's
// path relative to e's normalized path
func list(c *client.DomainSocketClient, e entry) ([]entry, error) {
	rawChildren, err := c.List(e.Path)
	if err != nil {
		return nil, err
	}
	children := make([]entry, len(rawChildren))
	for i, rawChild := range rawChildren {
		children[i] = entry{
			Entry:          rawChild,
			NormalizedPath: e.NormalizedPath + "/" + rawChild.CName,
		}
	}
	return children, nil
}
