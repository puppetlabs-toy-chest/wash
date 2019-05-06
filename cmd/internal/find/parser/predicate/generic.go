package predicate

// Generic represents a predicate on an interface{}
type Generic func(interface{}) bool

// And returns p1 && p2
func (p1 Generic) And(p2 Predicate) Predicate {
	return Generic(func(v interface{}) bool {
		return p1(v) && (p2.(Generic))(v)
	})
}

// Or returns p1 || p2
func (p1 Generic) Or(p2 Predicate) Predicate {
	return Generic(func(v interface{}) bool {
		return p1(v) || (p2.(Generic))(v)
	})
}

// Negate returns Not(p1)
func (p1 Generic) Negate() Predicate {
	return Generic(func(v interface{}) bool {
		return !p1(v)
	})
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 Generic) IsSatisfiedBy(v interface{}) bool {
	return p1(v)
}

// GenericParser is a type that parses Generic predicates
type GenericParser func(tokens []string) (Generic, []string, error)

// Parse parses a Generic predicate from the given input.
func (parser GenericParser) Parse(tokens []string) (Predicate, []string, error) {
	return parser(tokens)
}