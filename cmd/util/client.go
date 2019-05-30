package cmdutil

import (
	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/cmd/internal/config"
)

// NewClient returns a new Wash client for the given subcommand.
// Tests can set NewClient to a stub that returns a mock client.
var NewClient = func() client.Client {
	return client.ForUNIXSocket(config.Socket)
}