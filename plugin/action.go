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

var listAction = newAction("list", "Group")
// ListAction represents the list action
func ListAction() Action {
	return listAction
}

var readAction = newAction("read", "Readable")
// ReadAction represents the read action
func ReadAction() Action {
	return readAction
}

var streamAction = newAction("stream", "Streamable")
// StreamAction represents the stream action
func StreamAction() Action {
	return streamAction
}

var execAction = newAction("exec", "Execable")
// ExecAction represents the exec action
func ExecAction() Action {
	return execAction
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
	case *externalPluginRoot:
		return t.supportedActions
	case *externalPluginEntry:
		return t.supportedActions
	default:
		actions := make([]string, 0)

		// We could use reflection to simplify this. In fact, a previous version
		// of the code did do that. The reason we removed it was b/c type assertion's
		// a lot faster, and the resulting code isn't that bad, if a little verbose.
		if _, ok := entry.(Group); ok {
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

		return actions
	}
}
