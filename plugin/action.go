package plugin

// Action represents a Wash action.
type Action struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

func newAction(name string, protocol string) Action {
	return Action{
		Name:     name,
		Protocol: protocol,
	}
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

// ListAction represents the list action
var ListAction = newAction("list", "Group")

// ReadAction represents the read action
var ReadAction = newAction("read", "Readable")

// StreamAction represents the stream action
var StreamAction = newAction("stream", "Pipe")

// ExecAction represents the exec action
var ExecAction = newAction("exec", "Execable")

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
			actions = append(actions, ListAction.Name)
		}
		if _, ok := entry.(Readable); ok {
			actions = append(actions, ReadAction.Name)
		}
		if _, ok := entry.(Pipe); ok {
			actions = append(actions, StreamAction.Name)
		}
		if _, ok := entry.(Execable); ok {
			actions = append(actions, ExecAction.Name)
		}

		return actions
	}
}
