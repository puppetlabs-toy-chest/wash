package apitypes

import (
	"time"
)

// ExecOptions are options that can be passed as part of an Exec call.
// These are not identical to plugin.ExecOptions because initially the API only
// supports receiving a string of input, not a reader.
type ExecOptions struct {
	Input string `json:"input"`
}

// ExecBody encapsulates the payload for a call to a plugin's Exec function
type ExecBody struct {
	Cmd  string      `json:"cmd"`
	Args []string    `json:"args"`
	Opts ExecOptions `json:"opts"`
}

// ExecPacketType identifies the packet type.
type ExecPacketType = string

// Enumerates packet types.
const (
	Stdout   ExecPacketType = "stdout"
	Stderr   ExecPacketType = "stderr"
	Exitcode ExecPacketType = "exitcode"
)

// ExecPacket is a single packet of results from an exec.
// If TypeField is Stdout or Stderr, Data will be a string.
// If TypeField is Exitcode, Data will be an int (or float64 if deserialized from JSON).
type ExecPacket struct {
	TypeField ExecPacketType `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Data      interface{}    `json:"data"`
	Err       *ErrorObj      `json:"error"`
}
