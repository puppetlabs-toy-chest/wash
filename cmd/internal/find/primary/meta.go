package primary

import (
    "github.com/puppetlabs/wash/cmd/internal/find/types"
    "github.com/puppetlabs/wash/cmd/internal/find/primary/meta"
)

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

ObjectPredicate     => EmptyPredicate | ‘.’ Key OAPredicate

ArrayPredicate      => EmptyPredicate        |
                       ‘[' ? ‘]’ OAPredicate |
                       ‘[' * ‘]’ OAPredicate |
                       ‘[' N ‘]’ OAPredicate |

OAPredicate         => Predicate | "(" PredicateExpression ")"

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
var metaPrimary = Parser.add(&primary{
    tokens: []string{"-meta", "-m"},
    parseFunc: meta.Parse,
    optionsSetter: func(opts *types.Options) {
        if !opts.IsSet(types.MaxdepthFlag) {
            // The `meta` primary's a specialized filter. It should only be used
            // if a user needs to filter on something that isn't in plugin.EntryAttributes
            // (e.g. like an EC2 instance tag, a Docker container's image, etc.). Thus, it
            // wouldn't make sense for `wash find` to recurse when someone's using the `meta`
            // primary since it is likely that siblings or children will have a different meta
            // schema. For example, if we're filtering EC2 instances based on a tag, then `wash find`
            // shouldn't recurse down into the EC2 instance's console output + metadata.json files
            // because those entries don't have tags and, even if they did, they'd likely be under a
            // different key (e.g. like "Labels" for Docker containers). Thus to avoid the unnecessary
            // recursion, we default maxdepth to 1 if the flag was not set by the user. Note that users
            // who want to recurse down into subdirectories can just specify the maxdepth option, which
            // is useful when running `wash find` inside a directory whose entries and subdirectory entries
            // all have the same `meta` schema (e.g. like in an S3 bucket).
            opts.Maxdepth = 1
        }
    },
})
