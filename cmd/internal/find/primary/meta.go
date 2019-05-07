package primary

import "github.com/puppetlabs/wash/cmd/internal/find/primary/meta"

/*
metaPrimary         => (-meta|-m) Expression

Expression          => EmptyPredicate | KeySequence PredicateExpression
EmptyPredicate      => -empty

KeySequence         => '.' Key Tail
Key                 => [ ^.[ ] ]+ (i.e. one or more cs that aren't ".", "[", or "]")
Tail                => '.' Key Tail   |
                       ‘[' ? ‘]’ Tail |
                       '[' * ']' Tail |
                       '[' N ']' Tail |
                       ""

PredicateExpression => (See the comments of expression.Parser#Parse)

Predicate           => ObjectPredicate     |
                       ArrayPredicate      |
                       PrimitivePredicate

ObjectPredicate     => EmptyPredicate | ‘.’ Key Predicate

ArrayPredicate      => EmptyPredicate      |
                       ‘[' ? ‘]’ Predicate |
                       ‘[' * ‘]’ Predicate |
                       ‘[' N ‘]’ Predicate |

PrimitivePredicate  => NullPredicate       |
                       ExistsPredicate     |
                       BooleanPredicate    |
                       NumericPredicate    |
                       TimePredicate       |
                       StringPredicate

NullPredicate       => -null
ExistsPredicate     => -exists
BooleanPredicate    => -true | -false

NumericPredicate    => (+|-)? Number
Number              => N | '{' N '}' | numeric.SizeRegex

TimePredicate       => (+|-)? Duration
Duration            => numeric.DurationRegex | '{' numeric.DurationRegex '}'

StringPredicate     => [^-].*

N                   => \d+ (i.e. some number > 0)
*/
//nolint
var metaPrimary = Parser.newPrimary([]string{"-meta", "-m"}, meta.Parse)
