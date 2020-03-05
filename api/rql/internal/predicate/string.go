package predicate

import (
	"fmt"
	"regexp"

	"github.com/gobwas/glob"
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/matcher"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
)

/*
These are the individual string predicates
*/

func StringGlob(g string) rql.StringPredicate {
	return &stringGlob{
		gStr: g,
		g:    glob.MustCompile(g),
	}
}

type stringGlob struct {
	gStr string
	g    glob.Glob
}

func (p *stringGlob) Marshal() interface{} {
	return []interface{}{"glob", p.gStr}
}

func (p *stringGlob) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value("glob"))(input) {
		return errz.MatchErrorf("must be formatted as [\"glob\", <glob>]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as [\"glob\", <glob>]")
	}
	if len(array) < 2 {
		return fmt.Errorf("must be formatted as [\"glob\", <glob>] (missing the glob)")
	}
	globStr, ok := array[1].(string)
	if !ok {
		return fmt.Errorf("glob must be a string")
	}
	g, err := glob.Compile(globStr)
	if err != nil {
		return fmt.Errorf("invalid glob %v: %w", globStr, err)
	}
	p.gStr = globStr
	p.g = g
	return nil
}

func (p *stringGlob) EvalString(str string) bool {
	return p.g.Match(str)
}

var _ = rql.StringPredicate(&stringGlob{})

func StringRegex(r *regexp.Regexp) rql.StringPredicate {
	return &stringRegex{
		r: r,
	}
}

type stringRegex struct {
	r *regexp.Regexp
}

func (p *stringRegex) Marshal() interface{} {
	return []interface{}{"regex", p.r.String()}
}

func (p *stringRegex) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value("regex"))(input) {
		return errz.MatchErrorf("must be formatted as [\"regex\", <regex>]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as [\"regex\", <regex>]")
	}
	if len(array) < 2 {
		return fmt.Errorf("must be formatted as [\"regex\", <regex>] (missing the regex)")
	}
	regexStr, ok := array[1].(string)
	if !ok {
		return fmt.Errorf("regex must be a string")
	}
	r, err := regexp.Compile(regexStr)
	if err != nil {
		return fmt.Errorf("invalid regex %v: %w", regexStr, err)
	}
	p.r = r
	return nil
}

func (p *stringRegex) EvalString(str string) bool {
	return p.r.MatchString(str)
}

var _ = rql.StringPredicate(&stringRegex{})

func StringEqual(s string) rql.StringPredicate {
	return &stringEqual{
		s: s,
	}
}

type stringEqual struct {
	s string
}

func (p *stringEqual) Marshal() interface{} {
	return []interface{}{"=", p.s}
}

func (p *stringEqual) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value("="))(input) {
		return errz.MatchErrorf("must be formatted as [\"=\", <str>]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as [\"=\", <str>]")
	}
	if len(array) < 2 {
		return fmt.Errorf("must be formatted as [\"=\", <str>] (missing the string)")
	}
	s, ok := array[1].(string)
	if !ok {
		return fmt.Errorf("must provide a string")
	}
	p.s = s
	return nil
}

func (p *stringEqual) EvalString(str string) bool {
	return str == p.s
}

var _ = rql.StringPredicate(&stringEqual{})

/*
This is the main string predicate type
*/

type stringP struct {
	internal.NonterminalNode
}

func String() rql.StringPredicate {
	p := &stringP{
		NonterminalNode: internal.NewNonterminalNode(
			StringGlob(""),
			StringRegex(nil),
			StringEqual(""),
		),
	}
	p.SetMatchErrMsg("must be formatted as either [\"glob\", <glob>], [\"regex\", <regex>], or [\"=\", <str>]")
	return p
}

func (p *stringP) EvalString(str string) bool {
	return p.MatchedNode().(rql.StringPredicate).EvalString(str)
}

/*
This is the string predicate type that's also a value predicate. We make
it take an rql.StringPredicate instead of stringP so that it can be used
by parsers
*/

type stringValue struct {
	primitiveValueBase
	rql.StringPredicate
}

func (p *stringValue) Marshal() interface{} {
	return []interface{}{"string", p.StringPredicate.Marshal()}
}

func (p *stringValue) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value("string"))(input) {
		return errz.MatchErrorf("must be formatted as [\"string\", NPE StringPredicate]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as [\"string\", NPE StringPredicate]")
	}
	if len(array) < 2 {
		return fmt.Errorf("must be formatted as [\"string\", NPE StringPredicate] (missing the NPE StringPredicate)")
	}
	if err := p.StringPredicate.Unmarshal(array[1]); err != nil {
		return fmt.Errorf("error unmarshalling the NPE StringPredicate: %w", err)
	}
	return nil
}

func (p *stringValue) EvalValue(v interface{}) bool {
	str, ok := v.(string)
	return ok && p.EvalString(str)
}

func StringValue(p rql.StringPredicate) rql.ValuePredicate {
	s := &stringValue{
		StringPredicate: p,
	}
	s.primitiveValueBase = newPrimitiveValue(s)
	return s
}

func StringValueGlob(g string) rql.ValuePredicate {
	return StringValue(StringGlob(g))
}

func StringValueRegex(r *regexp.Regexp) rql.ValuePredicate {
	return StringValue(StringRegex(r))
}

func StringValueEqual(str string) rql.ValuePredicate {
	return StringValue(StringEqual(str))
}

var _ = meta.ValuePredicate(&stringValue{})
