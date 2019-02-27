package plugin

import (
	"reflect"
)

type Action struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	// Make sure this is an unexported field
	// so that the JSON encoder does not accidentally
	// marshal it
	protocolTypeObj reflect.Type
}

func newAction(name string, protocol interface{}) *Action {
	action := &Action{
		Name:            name,
		protocolTypeObj: reflect.TypeOf(protocol).Elem(),
	}
	action.Protocol = action.protocolTypeObj.Name()

	return action
}

// TODO: Could optimize this at some point
func (a *Action) IsSupportedOn(entry Entry) bool {
	switch t := entry.(type) {
	case *ExternalPluginRoot:
		return t.supportsAction(a)
	case *ExternalPluginEntry:
		return t.supportsAction(a)
	default:
		return reflect.TypeOf(entry).Implements(a.protocolTypeObj)
	}
}

// The(*Resource)(nil) trick was adapted from
// https://stackoverflow.com/a/7855298
var MetadataAction = newAction("metadata", (*Resource)(nil))
var ListAction = newAction("list", (*Group)(nil))
var ReadAction = newAction("read", (*Readable)(nil))
var StreamAction = newAction("stream", (*Pipe)(nil))
var ExecAction = newAction("exec", (*Execable)(nil))

var allActions = []*Action{
	MetadataAction,
	ListAction,
	ReadAction,
	StreamAction,
	ExecAction,
}

func SupportedActionsOf(entry Entry) []string {
	switch t := entry.(type) {
	case *ExternalPluginRoot:
		return t.supportedActions
	case *ExternalPluginEntry:
		return t.supportedActions
	default:
		actions := make([]string, 0)
		for _, action := range allActions {
			if action.IsSupportedOn(entry) {
				actions = append(actions, action.Name)
			}
		}

		return actions
	}
}
