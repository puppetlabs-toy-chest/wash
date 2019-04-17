package grammar

import "github.com/puppetlabs/wash/cmd/internal/find/types"

// Atom represents an atom in `wash find`'s expression grammar.
type Atom struct {
	Tokens []string
	// tokens[0] will always include the atom's token that the user
	// passed-in
	Parse func(tokens []string) (types.Predicate, []string, error)
}

// Atoms is a map of <token> => <atom>. This is populated by newAtom.
var Atoms = make(map[string]*Atom)

// NewAtom creates a new atom. When creating a new atom with this function
// via a package variable assignment, be sure to comment nolint above the
// variable so that CI does not mark it as unused. See notOp in find/operators.go
// for an example.
func NewAtom(tokens []string, parse func(tokens []string) (types.Predicate, []string, error)) *Atom {
	a := &Atom{
		Tokens: tokens,
		Parse:  parse,
	}
	for _, t := range tokens {
		Atoms[t] = a
	}
	return a
}
