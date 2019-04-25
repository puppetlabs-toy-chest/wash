package primary

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/cmd/internal/find/grammar"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// actionPrimary => <action>
//nolint
var actionPrimary = grammar.NewAtom([]string{"-action"}, func(tokens []string) (types.Predicate, []string, error) {
	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("-action: requires additional arguments")
	}
	validActions := plugin.Actions()
	action, ok := validActions[tokens[0]]
	if !ok {
		// User's querying an invalid action, so return an error.
		validActionsArray := make([]string, 0, len(validActions))
		for actionName := range validActions {
			validActionsArray = append(validActionsArray, actionName)
		}
		validActionsStr := strings.Join(validActionsArray, ", ")
		return nil, nil, fmt.Errorf("-action: %v is an invalid action. Valid actions are %v", tokens[0], validActionsStr)
	}
	p := func(e types.Entry) bool {
		return e.Supports(action)
	}
	return p, tokens[1:], nil
})
