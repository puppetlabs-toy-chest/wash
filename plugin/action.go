package plugin

// Action represents a Wash action.
type Action struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

var actions = make(map[string]Action)

func newAction(name string, protocol string) Action {
	a := Action{
		Name:     name,
		Protocol: protocol,
	}
	actions[a.Name] = a
	return a
}

// IsSupportedOn returns true if the action's supported
// on the specified entry, false otherwise.
func (a Action) IsSupportedOn(entry Entry) bool {
	for _, action := range SupportedActionsOf(entry) {
		if a.Name == action {
			return true
		}
	}

	return false
}

var listAction = newAction("list", "Parent")
var readAction = newAction("read", "Readable")
var streamAction = newAction("stream", "Streamable")
var execAction = newAction("exec", "Execable")
var deleteAction = newAction("delete", "Deletable")
var signalAction = newAction("signal", "Signalable")

// ListAction represents the list action
func ListAction() Action {
	return listAction
}

// ReadAction represents the read action
func ReadAction() Action {
	return readAction
}

// StreamAction represents the stream action
func StreamAction() Action {
	return streamAction
}

// ExecAction represents the exec action
func ExecAction() Action {
	return execAction
}

// DeleteAction represents the delete action
func DeleteAction() Action {
	return deleteAction
}

// SignalAction represents the signal action
func SignalAction() Action {
	return signalAction
}

// Actions returns all of the available Wash actions as a map
// of <action_name> => <action_object>.
func Actions() map[string]Action {
	// We create a clone of the actions map so that callers won't
	// be able to modify it.
	mp := make(map[string]Action)
	for k, v := range actions {
		mp[k] = v
	}
	return mp
}

// SupportedActionsOf returns all of the given
// entry's supported actions.
func SupportedActionsOf(entry Entry) []string {
	switch t := entry.(type) {
	case externalPlugin:
		var supportedActions []string
		for _, method := range t.supportedMethods() {
			// This ensures that we don't return methods like "metadata"/"schema",
			// which are not valid Wash actions.
			if _, ok := actions[method]; ok {
				supportedActions = append(supportedActions, method)
			}
		}
		return supportedActions
	default:
		actions := make([]string, 0)

		// We could use reflection to simplify this. In fact, a previous version
		// of the code did do that. The reason we removed it was b/c type assertion's
		// a lot faster, and the resulting code isn't that bad, if a little verbose.
		if _, ok := entry.(Parent); ok {
			actions = append(actions, ListAction().Name)
		}
		if _, ok := entry.(Readable); ok {
			actions = append(actions, ReadAction().Name)
		}
		if _, ok := entry.(Streamable); ok {
			actions = append(actions, StreamAction().Name)
		}
		if _, ok := entry.(Execable); ok {
			actions = append(actions, ExecAction().Name)
		}
		if _, ok := entry.(Deletable); ok {
			actions = append(actions, DeleteAction().Name)
		}
		if _, ok := entry.(Signalable); ok {
			actions = append(actions, SignalAction().Name)
		}

		return actions
	}
}
