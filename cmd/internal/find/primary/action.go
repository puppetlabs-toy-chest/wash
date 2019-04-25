package primary

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/cmd/internal/find/grammar"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// actionPrimary => <action> (, <action>)?
//nolint
var actionPrimary = grammar.NewAtom([]string{"-action"}, func(tokens []string) (types.Predicate, []string, error) {
	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("-action: requires additional arguments")
	}
	// Store the queried actions in a map for faster lookup.
	queriedActions := make(map[string]struct{})
	validActions := plugin.Actions()
	for i, a := range strings.Split(tokens[0], ",") {
		if a == "" {
			if i == 0 {
				return nil, nil, fmt.Errorf("expected an action before ','")
			} else {
				return nil, nil, fmt.Errorf("expected an action after ','")
			}
		}
		if _, ok := validActions[a]; !ok {
			// User's querying an invalid action, so return an error.
			validActionsArray := make([]string, 0, len(validActions))
			for actionName := range validActions {
				validActionsArray = append(validActionsArray, actionName)
			}
			validActionsStr := strings.Join(validActionsArray, ", ")
			return nil, nil, fmt.Errorf("-action: %v is an invalid action. Valid actions are %v", a, validActionsStr)
		}
		queriedActions[a] = struct{}{}
	}

	p := func(e types.Entry) bool {
		for _, a := range e.Actions {
			if _, ok := queriedActions[a]; ok {
				return true
			}
		}
		return false
	}
	return p, tokens[1:], nil
})
