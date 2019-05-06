package predicate

// Predicate is an interface representing a predicate.
// It is used to generalize the expression parsing logic
// used by the top-level parser and the meta primary.
//
// TODO: This interface is here because Go 1 doesn't support
// generics. Go 2 is planning to introduce some basic generic
// support, so we should re-evaluate this approach once it is
// released.
type Predicate interface {
	And(Predicate) Predicate
	Or(Predicate) Predicate
	Negate() Predicate
}