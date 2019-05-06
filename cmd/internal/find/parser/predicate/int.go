package predicate

// Int represents a predicate on an integer
type Int func(int64) bool

// And returns p1 && p2
func (p1 Int) And(p2 Predicate) Predicate {
	return Int(func(n int64) bool {
		return p1(n) && (p2.(Int))(n)
	})
}

// Or returns p1 || p2
func (p1 Int) Or(p2 Predicate) Predicate {
	return Int(func(n int64) bool {
		return p1(n) || (p2.(Int))(n)
	})
}

// Negate returns Not(p1)
func (p1 Int) Negate() Predicate {
	return Int(func(n int64) bool {
		return !p1(n)
	})
}

// IntParser is a type that parses Int predicates
type IntParser func(tokens []string) (Int, []string, error)

// Parse parses an Int predicate from the given input.
func (parser IntParser) Parse(tokens []string) (Predicate, []string, error) {
	return parser(tokens)
}