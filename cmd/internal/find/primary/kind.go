package primary

import (
	"fmt"
	"strings"

	"github.com/gobwas/glob"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// Kind is the kind primary
//
// kindPrimary => -kind ShellPattern
//nolint
var Kind = Parser.add(&Primary{
	Description:         "Returns true if the entry's kind matches pattern",
	DetailedDescription: kindDetailedDescription,
	name:                "kind",
	args:                "pattern",
	shortName:           "k",
	parseFunc: func(tokens []string) (types.EntryPredicate, []string, error) {
		if len(tokens) == 0 {
			return nil, nil, fmt.Errorf("requires additional arguments")
		}
		g, err := glob.Compile(tokens[0])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid pattern: %v", err)
		}
		return kindP(g, false), tokens[1:], nil
	},
})

func kindP(g glob.Glob, negated bool) types.EntryPredicate {
	p := kindPredicate{
		EntryPredicate: types.ToEntryP(func(e types.Entry) bool {
			// kind is a schema predicate, so the entry predicate should
			// always return true
			return true
		}),
		g: g,
	}
	p.SetSchemaP(types.ToEntrySchemaP(func(s *types.EntrySchema) bool {
		segments := strings.SplitN(s.Path(), "/", 2)
		if len(segments) <= 1 {
			// s is the stree root, so always return false here.
			return false
		}
		kind := segments[1]
		if !g.Match(kind) {
			return negated
		}
		// g matches kind
		if negated {
			return false
		}
		return true
	}))
	p.RequireSchema()
	return p
}

// The separate type's necessary to implement proper Negation semantics.
type kindPredicate struct {
	types.EntryPredicate
	g       glob.Glob
	negated bool
}

func (p kindPredicate) Negate() predicate.Predicate {
	return kindP(p.g, !p.negated)
}

const kindDetailedDescription = `
(-k|-kind) pattern

The kind primary constructs a predicate on the entry's kind, where the entry's
kind is its schema path but without the <root_label>. It will return true if
the entry's kind matches pattern, false otherwise. Note that the kind primary
will always return false for schema-less entries.

An entry's schema path is constructed as <root_label>/<parent1_label>/.../<label>,
where <root_label> is the label of the stree root(s). The stree root(s) are the
path(s) passed into find. Thus, an entry's kind is <parent1_label>/.../<label>.

The kind primary's usage is best illustrated by some examples.

EXAMPLES:

find docker -kind '*container'
find docker -k '*container'
    This prints out all Docker containers. The '*' is there because a Docker
    container could be nested several levels deep in the Docker plugin. For
    example, at the time this help-text was written, a Docker container's
    kind is "containers/container" (where "docker" is the stree root since that
    is the passed-in path). There is no reason why the kind couldn't change to
    something like "containers/state/container". Thus, prefixing a kind pattern
    with '*' makes the query less dependent on a plugin's hierarchy (and hence more
    expressive).

find docker aws -kind '*metadata.json'
find docker aws -k '*metadata.json'
    This prints out all the metadata.json entries in the docker and aws plugins. Here, the
    stree root(s) are the "docker" and "aws" entries' schemas. Note that this is a
    conceptual example. It is meant to showcase how the kind primary works when multiple
    paths are passed into find.

find aws/demo -kind '*s3*bucket'
find aws/demo -k '*s3*bucket'
    This prints out all S3 buckets in the demo profile.

find docker -kind '*volumes*dir'
find docker -k '*volumes*dir'
    This prints out all Docker volume directories.

find docker -kind '*container' -mtime -1h
find docker -k '*container' -mtime -1h
    This prints out all Docker containers that were modified within the last hour. Note that
    without the kind primary, find would have visited all entries in the Docker plugin.

find -kind 'docker/*container' -o -k 'aws/*ec2*instance' -mtime -1h
find -k 'docker/*container' -o -k 'aws/*ec2*instance' -mtime -1h
    This prints out all Docker containers and EC2 instances that were modified within the
    last hour.

NOTE: You can use 'stree <path>' to determine an entry's kind. For example, if <path>
is "docker", then

stree docker
docker
├── containers
│   └── [container]
│       ├── log
│       ├── metadata.json
│       └── fs
│           ├── [dir]
│           │   ├── [dir]
│           │   └── [file]
│           └── [file]
└── volumes
    └── [volume]
        ├── [dir]
        │   ├── [dir]
        │   └── [file]
		└── [file]

We see from the output that a Docker container's kind is "containers/container".
Similarly, a Docker volume directory's kind is "volumes/volume/dir" OR
"volumes/volume/dir/dir". The latter comes from the definition -- a volume directory
has more than one possible schema path and hence more than one possible kind. This
discrepancy is one limitation of the current schema representation. We may address
this discrepancy in a future Wash release; however, leaving it as-is for now simplifies
the plugin and entry schema interface while still delivering the same functionality.
`
