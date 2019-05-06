package primary

import "github.com/puppetlabs/wash/cmd/internal/find/primary/meta"

/*
metaPrimary        => (-meta|-m) ObjectPredicate
ObjectPredicate    => EmptyPredicate | ‘.’ Key Predicate
EmptyPredicate     => -empty
Key                => [ ^.[ ] ]+ (i.e. one or more cs that aren't ".", "[", or "]")

Predicate          => ObjectPredicate     |
                      ArrayPredicate      |
                      PrimitivePredicate

ArrayPredicate     => EmptyPredicate      |
                      ‘[' ? ‘]’ Predicate |
                      ‘[' * ‘]’ Predicate |
                      ‘[' N ‘]’ Predicate |

PrimitivePredicate => NullPredicate       |
                      ExistsPredicate     |
                      BooleanPredicate    |
                      NumericPredicate    |
                      TimePredicate       |
                      StringPredicate

NullPredicate      => -null
ExistsPredicate    => -exists
BooleanPredicate   => -true | -false

NumericPredicate   => (+|-)? Number
Number             => N | '{' N '}' | numeric.SizeRegex

TimePredicate      => (+|-)? Duration
Duration           => numeric.DurationRegex | '{' numeric.DurationRegex '}'

StringPredicate    => [^-].*

N                  => \d+ (i.e. some number > 0)
*/
//nolint
var metaPrimary = Parser.newPrimary([]string{"-meta", "-m"}, meta.Parse)
