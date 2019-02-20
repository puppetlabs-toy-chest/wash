package api

import (
	"reflect"

	"github.com/puppetlabs/wash/plugin"
)

type action struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	// Make sure this is an unexported field
	// so that the JSON encoder does not accidentally
	// marshal it
	protocolTypeObj reflect.Type
}

func newAction(name string, protocol interface{}) *action {
	action := &action{
		Name:            name,
		protocolTypeObj: reflect.TypeOf(protocol).Elem(),
	}
	action.Protocol = action.protocolTypeObj.Name()

	return action
}

func (a *action) isSupportedOn(entry plugin.Entry) bool {
	return reflect.TypeOf(entry).Implements(a.protocolTypeObj)
}

// The(*plugin.Resource)(nil) trick was adapted from
// https://stackoverflow.com/a/7855298
var metadataAction = newAction("metadata", (*plugin.Resource)(nil))
var listAction = newAction("list", (*plugin.Group)(nil))
var readAction = newAction("read", (*plugin.Readable)(nil))
var streamAction = newAction("stream", (*plugin.Pipe)(nil))
var execAction = newAction("exec", (*plugin.Execable)(nil))

var allActions = []*action{
	metadataAction,
	listAction,
	readAction,
	streamAction,
	execAction,
}

func supportedActionsOf(entry plugin.Entry) []string {
	actions := make([]string, 0)
	for _, action := range allActions {
		if action.isSupportedOn(entry) {
			actions = append(actions, action.Name)
		}
	}

	return actions
}
