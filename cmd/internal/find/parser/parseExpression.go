package parser

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/expression"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

/*
See the comments of expression.Parser#Parse for the grammar. Substitute
Predicate with Primary.
*/
func parseExpression(tokens []string) (types.EntryPredicate, error) {
	if len(tokens) == 0 {
		// tokens is empty, meaning the user did not provide an expression
		// to `wash find`. Thus, we default to a predicate that always returns
		// true.
		return func(e types.Entry) bool {
			return true
		}, nil
	}
	parser := expression.NewParser(primary.Parser)
	p, tks, err := parser.Parse(tokens)
	if err != nil {
		if tkErr, ok := err.(expression.UnknownTokenError); ok {
			err = fmt.Errorf("%v: unknown primary or operator", tkErr.Token)
		}
		return nil, err
	}
	if len(tks) != 0 {
		// This should never happen, but better safe than sorry
		msg := fmt.Sprintf("parser.parseExpression(): returned a non-empty set of tokens: %v", tks)
		panic(msg)
	}
	return p.(types.EntryPredicate), nil
}

// OperandsTable returns a table containing all of `wash find`'s available
// operands
func OperandsTable() *cmdutil.Table {
	return cmdutil.NewTable(
		[]string{"Operands:",    ""},
		[]string{"  ( e )",      "Parentheses operator. Evaluates to true if e evaluates to true"},
		[]string{"  !, -not e",  "Logical NOT operator. Evaluates to true if e evaluates to false"},
		[]string{"  e1 -and e2", "Logical AND operator. Evalutes to true if both e1 and e2 are true"},
		[]string{"  e1 -a e2",   ""},
		[]string{"  e1 e2",      ""},
		[]string{"  e1 -or e2",  "Logical OR operator. Evalutes to true if either e1 or e2 are true"},
		[]string{"  e1 -o e2",   ""},
	)
}

func isPartOfExpression(arg string) bool {
	parser := expression.NewParser(primary.Parser)
	return arg == "--" || parser.IsOp(arg) || primary.Parser.IsPrimary(arg)
}

// ExpressionSyntaxDescription describes `wash find`'s expression syntax.
const ExpressionSyntaxDescription = `
An expression evaluates to a predicate that is applied on a specific set
of entries. By default, this set consists of all traversed entries,
including the specified path. You can use the mindepth/maxdepth options
to restrict this set.

Expressions consist of primaries and operators. Primaries yield a predicate
that's set on a specific property of the entry (e.g. name, path, ctime),
while the operators combine these "primary predicates" to form more
powerful, expressive predicates. Use "wash find -h" to view the available
primaries, and "wash find -h <primary>" to view a detailed description of the
specified primary, including examples of how to use it.

The rest of this section discusses the operators, their precedence, and
includes various examples of valid expressions.

OPERATORS:
The operators are listed in order of decreasing precedence.

( expression )
    Evaluates to true if the parenthesized expression evaluates to true.

! expression
-not expression
    Unary NOT operator. Evaluates to true if the expression is false.

expression -and expression
expression -a expression
expression expression
    Logical AND operator. Evaluates to true if both expressions evaluate
    to true. Note that the second expression is not evaluated if the
    first expression is false.

expression -or expression
expression -o expression
    Logical OR operator. Evaluates to true if both expressions evaluate to
    true. Note that the second expression is not evaluated if the first
    expression is true.

EXAMPLES:
The following examples are shown as given to the shell. 

find . -name "*c"
    Print out all entries whose cname ends with a "c"

find . \! -name "*c"
    Print out all entries whose cname does not end with a "c"

find . -name "*c" -mtime +1h
find . -name "*c" -a -mtime +1h
    Print out all entries whose cname ends with a "c" and
    whose last modification time was more than one hour ago.

find . -name "*c" -o -name "*d"
    Print out all entries whose cname ends with a "c" or a "d"

find . -daystart -name "*.log" -mtime 0
find . -daystart -name "*.log" -a -mtime 0
    Print out all log files that were updated today.

find . -path "docker*containers*" \( -name "*c" -o -name "*d" \) -a -mtime -1h
find . -path "docker*containers*" \( -name "*c" -o -name "*d" \) -mtime -1h
    Print out all the Docker containers whose cname ends with a "c"
    or a "d" that were modified within the last hour. Note that
    without the parentheses, the latter part of the expression would
    have been parsed as '-name "*c" OR ( -name "*d" AND -mtime -1h)',
    which is a completely different predicate.

find kubernetes -maxdepth -1 -daystart -m .status.startTime -{1d}
    Print out all the Kubernetes pods that started today. Note that the "maxdepth -1"
    option tells find to recurse (the meta primary turns this off by default).

find docker/containers -daystart -fullmeta -m .state.startedAt -{1d}
    Prints out all the Docker containers that were started today. Note that the
    "fullmeta" option is necessary because the "meta" attribute for a Docker container
    does not include the container's start time.

find ec2/instances -m .state.name running -a -m '.tags[?]' .key termination_date -a .value +0h
find ec2/instances -m .state.name running -m '.tags[?]' .key termination_date .value +0h
    Print out all the running EC2 instances whose termination_date tag expired.
    Note that the ".key termination_date -a .value +0h" portion of the example
    is parsed as part of the meta primary's expression, not the top level
    expression parser.
`
