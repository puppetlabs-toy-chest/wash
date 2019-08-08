package apitypes

import (
	"github.com/puppetlabs/wash/analytics"
)

// ScreenviewBody encapsulates the payload for a call to
// analytics.Client#Screenview
type ScreenviewBody struct {
	Name   string          `json:"name"`
	Params analytics.Params `json:"params"`
}
