package apitypes

import (
	"github.com/puppetlabs/wash/plugin"
	"time"
)

// ExecOptions are options that can be passed as part of an Exec call.
// These are not identical to plugin.ExecOptions because initially the API only
// supports receiving a string of input, not a reader.
type ExecOptions struct {
	// Input to pass on stdin when executing the command
	Input string `json:"input"`
}

// ExecBody encapsulates the payload for a call to a plugin's Exec function
//
// swagger:parameters executeCommand
type ExecBody struct {
	// Name of the executable to invoke
	//
	// in: body
	Cmd string `json:"cmd"`
	// Array of arguments to the executable
	//
	// in: body
	Args []string `json:"args"`
	// Additional execution options
	//
	// in: body
	Opts ExecOptions `json:"opts"`
}

// Enumerates packet types used by the API.
const (
	Stdout   plugin.ExecPacketType = plugin.Stdout
	Stderr   plugin.ExecPacketType = plugin.Stderr
	Exitcode plugin.ExecPacketType = "exitcode"
)

// ExecPacket is a single packet of results from an exec.
// If TypeField is Stdout or Stderr, Data will be a string.
// If TypeField is Exitcode, Data will be an int (or float64 if deserialized from JSON).
//
// swagger:response
type ExecPacket struct {
	TypeField plugin.ExecPacketType `json:"type"`
	Timestamp time.Time             `json:"timestamp"`
	Data      interface{}           `json:"data"`
	Err       *ErrorObj             `json:"error"`
}
