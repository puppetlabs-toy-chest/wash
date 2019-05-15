package parser

import (
	"flag"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/expression"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

func parseOptions(args []string) (types.Options, []string, error) {
	o := types.NewOptions()
	fs := o.FlagSet()

	// Find the index of the first non-option arg that FlagSet
	// doesn't know about. This is either "--", the special flag
	// termination symbol, or a primary/operator, which indicates
	// the beginning of a `wash find` expression. These non-option
	// args will be handled/processed in parseExpression. Note that
	// without this code, FlagSet would interpret "--" as the flag
	// termination symbol, which is bad because "--" is invalid
	// `wash find` syntax. It would also try to parse our primaries
	// and operators (like -true, -false, -not, -and) as options, which is
	// also bad.
	//
	// All other cases are properly handled by fs.Parse(). For example,
	// if args is of the form ["-mindepth", "0", "foo", "bar", "baz"],
	// then fs.Parse() will stop at "foo" so that parseExpression will
	// (correctly) receive the ["foo", "bar", "baz"] portion of the
	// arguments.
	var endIx int
	for _, arg := range args {
		if nonOptionArg(arg) {
			break
		}
		endIx++
	}

	// Parse the args
	err := fs.Parse(args[0:endIx])
	if err != nil {
		if err == flag.ErrHelp {
			// Parse the help option. Note that we cannot use Go's Value interface
			// to handle the parsing because that does not enable the "-help" and
			// "-help <primary>|syntax" usages of the flag. We could get it to work
			// as "-help" and "-help=<primary>|syntax", but that is inconsistent with
			// the other options like maxdepth (which allows both "-maxdepth 1" and
			// "-maxdepth=1"), and it introduces ambiguity between "-help" and
			// "-help=<primary>" when <primary> = true. The latter ambiguity is due to
			// Go's flag package treating "-help" as semantically equivalent to "-help=true".
			//
			// In general, Go's flag package does not allow one to treat a given
			// option as a Boolean option with an optional value without sacrificing
			// some syntactic consistency by requiring the value to be specified as
			// "<flag>=<value>" instead of both "<flag>=<value>" and "<flag> <value>".
			o.Help.Requested = true	
			// args contains the part of args after the help flag.
			args = fs.Args()
			if len(args) > 0 {
				arg := args[0]
				// The "-" check is to make sure it doesn't conflict with
				// another option or a `wash find` primary
				if len(arg) > 0 && arg[0] != '-' {
					o.Help.HasValue = true
					if arg == "syntax" {
						o.Help.Syntax = true
					} else {
						o.Help.Primary = arg
					}
				}
			}
		}
		return o, nil, err
	}
	fs.Visit(func(f *flag.Flag) {
		o.MarkAsSet(f.Name)
		if f.Name == types.MaxdepthFlag && o.Maxdepth < 0 {
			o.Maxdepth = types.DefaultMaxdepth
		}
	})

	// Calculate the remaining args
	if endIx == len(args) {
		// This case includes the earlier ["-mindepth", "0", "foo", "bar", "baz"]
		// example. Here, calling fs.Args() would return ["foo", "bar", "baz"], which
		// are the remaining args
		args = fs.Args()
	} else {
		// args contained either "--", or an atom/binary op.
		args = args[endIx:]
	}

	return o, args, nil
}

func nonOptionArg(arg string) bool {
	parser := expression.NewParser(primary.Parser)
	return arg == "--" || parser.IsOp(arg) || primary.Parser.IsPrimary(arg)
}
