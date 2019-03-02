package plugin

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

func (a Action) IsSupportedOn(entry Entry) bool {
	for _, action := range SupportedActionsOf(entry) {
		if a.Name == action {
			return true
		}
	}

	return false
}

var MetadataAction = newAction("metadata", "Resource")
var ListAction = newAction("list", "Group")
var ReadAction = newAction("read", "Readable")
var StreamAction = newAction("stream", "Pipe")
var ExecAction = newAction("exec", "Execable")

func SupportedActionsOf(entry Entry) []string {
	switch t := entry.(type) {
	case *ExternalPluginRoot:
		return t.supportedActions
	case *ExternalPluginEntry:
		return t.supportedActions
	default:
		actions := make([]string, 0)

		// We could use reflection to simplify this. In fact, a previous version
		// of the code did do that. The reason we removed it was b/c type assertion's
		// a lot faster, and the resulting code isn't that bad, if a little verbose.
		if _, ok := entry.(Resource); ok {
			actions = append(actions, MetadataAction.Name)
		}
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
