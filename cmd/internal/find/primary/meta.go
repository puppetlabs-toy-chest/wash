package primary

import (
	"github.com/puppetlabs/wash/cmd/internal/find/primary/meta"
)

/*
Meta is the meta primary

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
var Meta = Parser.add(&Primary{
	Description:         "Returns true if the entry's metadata satisfies the expression",
	DetailedDescription: metaDetailedDescription,
	name:                "meta",
	args:                "<expression>",
	shortName:           "m",
	parseFunc:           meta.Parse,
})

const metaDetailedDescription = `
The meta primary constructs a predicate on the entry's metadata. By
default, this is the meta attribute. If you'd like to construct the
predicate on the entry's full metadata, then set the "fullmeta" option.
Be careful when you do this, because find will make O(N) API requests
to retrieve this information (N = the number of visited entries). As a
precaution, the "fullmeta" option is only supported on entries with
schemas.

Meta is a specialized filter, so you should only use it if you need to
filter your entries on a property that isn't captured by the common Wash
attributes. For example, the meta primary can be used to filter EC2
instances on a specific tag. It can be used to filter Docker containers
with a specified label. In general, the meta primary can be used to filter
on any property that's specified in the entry's meta attribute. See the
EXAMPLES section for some interesting real-world examples.

NOTE: The meta primary should be used in conjunction with the kind primary.
The latter's useful for explicitly specifying the kind of entries you're
filtering on, which enables find to take full advantage of entry schema
optimizations. This is especially useful when the entry's schema doesn't
include metadata schemas (possible for external plugins). The kind primary's
also useful for making meta queries more expressive. See the EXAMPLES section
for more details on what this looks like.

NOTE: If your plugin's API is not subscription based (like AWS) or if
individual API requests are cheap, then feel free to always set the
"fullmeta" option for more complete filtering. This is very useful when
your plugin API's over a Unix socket.

NOTE: You can use the meta command to construct meta primary queries.
Here's how. First, find a representative entry that you'll be filtering.
For example, if you are filtering EC2 instances, then your representative
entry would be an EC2 instance. Next, invoke "meta <entry_path> -a" to
see that entry's meta attribute. Check the output to see if it contains
the properties you'd like to filter on. If yes, then construct the
predicate. If no, then invoke "meta <entry_path>" to see the entry's
full metadata, and check its output to see if it contains your properties.
If the properties are there, and the O(N) API requests made by "fullmeta"
aren't an issue, then construct the predicate and be sure to set the
"fullmeta" option when invoking "wash find". Otherwise, contact the plugin
author to see if they can add those properties to the entry's full metadata
or, preferably, the meta attribute.

NOTE: If the current meta primary predicates aren't enough to suit your
needs, then please file an issue or feel free to add one yourself!

USAGE:
(-m|-meta) (-empty | KeySequence PredicateExpression)

If -empty is specified, then returns true if the entry's meta attribute
is empty. Otherwise, returns true if the meta value specified by the key
sequence satisfies the given predicate expression.

KEY SEQUENCES:
A key sequence consists of a key token followed by zero or more "chunks".
A "chunk" is a key token OR an array token. Key tokens must be prefixed by
a ".", and cannot contain a ".", "[", or "]". Valid array tokens are "[?]",
"[*]", or "[n]", where n >= 0.

Key sequences are parsed as chunks, where each chunk represents either an
object predicate or an array predicate on the current meta value. A key
token chunk represents an object predicate, while an array token chunk
represents an array predicate. Each chunk is followed by a predicate, which
is either another chunk OR a predicate expression. Below are the semantics
for each chunk:

  .foo p
      Returns false if the current value is not an object. Otherwise, returns
      false if the value does not have a key matching 'foo'. A key matches
      'foo' if upcase(key) == 'FOO'. Otherwise, returns p(o[mkey]) where mkey
      is the first matching key.

  .[?] p
      Returns false if the current value is not an array. Otherwise, returns
      true if p returns true for some element in the array. Otherwise, returns
      false.

  .[*] p
      Returns false if the current value is not an array. Otherwise, returns
      true if p returns true for all elements in the array. Otherwise, returns
      false.

  .[n] p
      Returns false if the current value is not an array. Otherwise, returns
      false if the array has less than n elements. Otherwise, returns p(a[n]).

Below are some KeySequence examples. Note that 'p' represents the
predicate parsed by the PredicateExpression part of the input, while
'm' represents the entry's metadata. For brevity, assume that 'foo'
in queries of the form m['foo'] represents the first key in m that
matches 'foo'.

  .foo p
      Returns p(m['foo'])

  .foo.bar p
      Returns p(m['foo']['bar'])

  .foo[?] p
      Returns true if p returns true for some element in m['foo']

  .foo[?].bar p
      Returns true if some element in m['foo'] is an object o s.t.
      p(o['bar']) returns true.

PREDICATE EXPRESSIONS:
Predicate expression syntax is structurally identical to the top-level
expression syntax (type "wash find -h syntax" to get an overview of the
latter). Specifically, the "primaries" of predicate expressions consist of
either an object predicate, an array predicate, or a primitive predicate
(i.e. a "predicate") while the "operands" of predicate expressions are the
same as the operands in the top-level expression parser.

See the EXAMPLES section for predicate expression examples. The upcoming
sections describe each of the predicates.

OBJECT PREDICATE:
-empty                                         |
.key (Predicate | '(' PredicateExpression ')')

If -empty is specified, then returns true if the current meta value is an
empty object.

The latter part of the expression has the same semantics as the object
predicate described in the KEY SEQUENCES section. The only difference here
is that the "p" in ".foo p" is either a "predicate" OR a parenthesized
predicate expression.

Below are some examples of object predicates. Let "m" represent the value
that the predicate's being applied on.

  -empty
      Returns true if m is empty
    
  .foo +1
      Returns true if m['foo'] > 1

  .foo \( +1 -a -3 \)
      Returns true if m['foo'] > 1 AND m['foo'] < 3. In other words, this
      returns true if 1 < m['foo'] < 3.

  .foo.bar \( \! +1 \)
  .foo .bar \( \! +1 \)  
      Returns true if m['foo']['bar'] is NOT > 1. In other words, this returns
      true if m['foo']['bar'] <= 1. Also note that the "p" here is
      ".bar \( \! +1 \)". This example is meant to show that key sequences are
      are also supported by object predicates.

ARRAY PREDICATE:
-empty                                              |
'[' ? ']' (Predicate | '(' PredicateExpression ')') |
'[' * ']' (Predicate | '(' PredicateExpression ')') |
'[' n ']' (Predicate | '(' PredicateExpression ')') |

If -empty is specified, then returns true if the current meta value is an
empty array.

The latter part of the expression has the same semantics as the array
predicate described in the KEY SEQUENCES section. The only difference here
is that the "p" in "[?] p" is either a "predicate" OR a parenthesized
predicate expression.

Below are some examples of array predicates. Let "a" represent the value
that the predicate's being applied on.

  -empty
      Returns true if a is empty
    
  [?] +1
      Returns true if a has some element that's > 1

  [*] +1
      Returns true if all of a's elements are > 1

  [n] +1
      Returns true if a[n] > 1

  [?] \( +1 -a -3 \)
      Returns true if a has some element e s.t. e > 1 AND e < 3. In
      other words, this returns true if 1 < e < 3.

  [?].bar \( \! +1 \)
  [?] .bar \( \! +1 \)  
      Returns true if a has some element e s.t. e['bar'] is NOT > 1.
      In other words, this returns true if e['bar'] <= 1. Also note
      that the "p" here is ".bar \( \! +1 \)". This example is meant to
      show that key sequences are are also supported by array predicates.

PRIMITIVE PREDICATE:
NullPredicate    |
ExistsPredicate  |
BooleanPredicate |
NumericPredicate |
TimePredicate    |
StringPredicate

All of the above predicates are described in their own section.

NULL PREDICATE:
-null

Returns true if the current meta value is null. For example, ".foo -null"
returns true if m['foo'] is null.

EXISTS PREDICATE:
-exists

Returns true if the current meta value is not null. The exists predicate
is a useful way to check if an object has a specific key. For example,
".foo -exists" returns true if m['foo'] is not null, i.e. if m['foo']
exists.

BOOLEAN PREDICATE:
-true | -false

Boolean predicates always return false for a non-Boolean value. -true returns
true if the value is true, while -false returns true if the value is false.
For example, ".foo -true" returns true if m['foo'] is true, while ".foo -false"
returns true if m['foo'] is false.

NUMERIC PREDICATE:
[+|-]? N [ckMGTP] |
[+|-]? '{' N '}'

where N >= 0. Here are the semantics for numeric predicates. Let v be the
current meta value that's being compared. Then:
  * If v is not a number (i.e. not a float64 type), then the predicate returns
    false

  * If only N is specified, then the predicate returns v == N.

  * If +N is specified, then the predicate returns v > N

  * If -N is specified, then the predicate returns v < N

  * If N is suffixed with a unit, then v is compared to N scaled as:
        c        character (* 1)
        k        kibibytes (* 1024)
        M        mebibytes (* 1024 ^ 2)
        G        gibibytes (* 1024 ^ 3)
        T        tebibytes (* 1024 ^ 4)
        P        pebibytes (* 1024 ^ 5)
    Suffixing N with units is useful when v represents a size value (e.g. like
    the size of a VM's filesystem or the amount of memory that VM has).

  * If {N} is specified, then the predicate compares v with -N. {N} is useful
    for comparing negative numbers.

Below are some examples of numeric predicates

  1
      Returns true if v == 1

  {1}
      Returns true if v == -1

  +1
      Returns true if v > 1

  -1
      Returns true if v < 1

  +{1}
      Returns true if v > -1

  -{1}
      Returns true if v < -1

  1k
      Returns true if v = (1 * 1024)

  +1k
      Returns true if v > (1 * 1024)

  -1k
      Returns true if v < (1 * 1024)

And here's an example of a numeric predicate being used in conjunction
with an object predicate:

  .memory +1G
      Returns true if m['memory'] > 1 gibibyte (1 * 1024 ^ 3)

NOTE: Syntax like {1k} is currently not supported. If supported, this
would semantically mean (-1 * 1024). The reason we don't support it is
because negative size values do not make sense. If you do want to compare
against negative size values, you'll have to type out the full number.
For example, instead of {1k}, you'll need to type out {1024}.

If you find that you frequently need to compare against negative size values,
then please file a feature request so that we can add this support!

TIME PREDICATE:
[+|-]? N <smhdw>         |
[+|-]? '{' N <smhdw> '}'

where N >= 0. Note that units must be specified. Otherwise, there is no (simple)
way for the parser to distinguish between a numeric predicate and a time
predicate.

Here are the semantics for time predicates. Let v be the current meta value that's
being compared. Then:
  * If v does not represent a time value, then the predicate returns false. Examples
    of time values include stringified dates or unix seconds.

  * If Nu is specified, where Nu means N suffixed with a unit, then the difference "d"
    between the reference time and v will be compared to N scaled as: 
        s        second
        m        minute (60 seconds)
        h        hour   (60 minutes)
        d        day    (24 hours)
        w        week   (7 days) 
    The predicate returns "d" == Nu

  * If +Nu is specified, then the predicate returns "d" > Nu

  * If -Nu is specified, then the predicate returns "d" < Nu

  * If {Nu} is specified, then "d" is the difference between v and the reference time.
    Brackets are useful to distinguish "future" queries from "past" queries.

NOTE: If "d" < 0, then the predicate always returns false. "d" < 0 represents a time
mismatch (i.e. you are making a past query on a future time value or a future query on a
past time value). We impose this limitation to make it easier for people to reason about
time predicates. Otherwise, cases like "-1h" would return true for entries whose meta value
is a time value in the future despite "-1h" being mentally parsed as "less than one hour
ago", and thus interpreted as making sense for entries whose time values are in the past.

Below are some examples of time predicates.

  1s
      Returns true if v was exactly one second ago

  1h
      Returns true if v was exactly one hour ago

  +1h
      Returns true if v was more than one hour ago

  -1h
      Returns true if v was less than one hour ago

  +{1h}
      Returns true if v is more than one hour from now

  -{1h}
      Returns true if v is less than one hour from now

And here are some examples of time predicates being used in conjunction with
object predicates:

  .expiration_date +1h
      Returns true if m['expiration_date'] was more than one hour ago

  .expiration_date +{1h}
      Returns true if m['expiration_date'] is more than an hour from now

NOTE: As can be seen from the examples, time predicates only make sense when
prefixed with a "+" or a "-".

NOTE: As the expiration_date example shows, the "{}" distinguish a "future"
query from a "past" query. Future queries are a useful way of filtering out
entries that are going to expire within the next N minutes/hours/days/weeks
(e.g. security policies, tagged EC2 instances, user credentials, etc.)

NOTE: If the "daystart" option is set, then "-{1d}" returns true if the time
value is within the current day (i.e. today).

STRING PREDICATE:
Any input that doesn't begin with a "+", "-", "!", "(", or ")" is treated as
a string predicate. String predicates return false for any non-string values.
Otherwise, they return true if v == input, where v is the current meta value
that's being compared.

Below are some examples of string predicates:

  foo
      Returns true if v == foo 

  bar
      Returns true if v == bar

And here's an example of a string predicate being used in conjunction with
an object predicate:

  .owner Mike
      Returns true if m['owner'] == "Mike"

NOTE: String predicates are a bit limited right now. We plan on adding
regex/glob support for string predicates, and possibly a way for users to
specify quoted string literals. The latter's useful when the input begins with
one of the dis-allowed characters "+", "-", "!", "(", ")".

NEGATION SEMANTICS:
This section describes each of the aforementioned predicates' negation semantics.
The information here can be used to reason about input like "! .foo +1".

  OBJECT PREDICATE:  
  "! .foo p" == ".foo ! p"

  ARRAY PREDICATE:
  "! [?] p" == "[*] ! p"
  "! [*] p" == "[?] ! p"
  "! [n] p" == "[n] ! p"

  NULL PREDICATE:
  "! -null" == "-exists"

  EXISTS PREDICATE:
  "! -exists" == "-null"

  BOOLEAN PREDICATE:
  "! -true"  == "-false"
  "! -false" == "-true"

  NUMERIC PREDICATE:
  "! N"  == "+N" -o "-N"       (returns v != N)
  "! +N" == "-N" -o "N"        (returns v <= N)
  "! -N" == "+N" -o "N"        (returns v >= N)

  Note that here, N represents a number. For example, N could be specified as
  "1", "1k", or "{1}".

  TIME PREDICATE:
  "! 1h"    == "+1h" -o "-1h"          (returns "d" != 1h)
  "! +1h"   == "-1h" -o "1h"           (returns "d" <= 1h)
  "! -1h"   == "+1h" -o "1h"           (returns "d" >= 1h)

  where "d" = "FindStartTime - v". Similarly for "future" queries,

  "! {1h}"  == "+{1h}" -o "-{1h}"      (returns "d" != 1h)
  "! +{1h}" == "-{1h}" -o "{1h}"       (returns "d" <= 1h)
  "! -{1h}" == "+{1h}" -o "{1h}"       (returns "d" >= 1h)

  where "d" = "v - FindStartTime". Note that the above semantics imply that
  a negated time predicate will still return false if "d < 0" (i.e.
  time-mismatches still return false)

NOTE: The above semantics imply that the negated predicates still return false for
mis-typed values. For example, "! .foo p" returns false if v is not an object;
"! [?] p" returns false if v is not an array, etc.

EXAMPLES:
This section contains various, real-world examples of the meta primary's usage. If
you have an interesting example that you'd like to include here, then please feel
free to submit a PR!

In these examples, let "m" be the value of the entry's 'meta' attribute.

-meta '.tags[?]' .key termination_date -a .value +0h
-m '.tags[?]' .key termination_date -a .value +0h
    Returns true if m['tags'] has at least one object o s.t. o['key'] == termination_date
    and o['value'] has expired. In the real world, this example could be combined with the
    kind primary to filter out all EC2 instances whose termination_date tag expired. The
    expression would look something like
        find aws/demo -k '*ec2*instance' -m '.tags[?]' .key termination_date -a .value +0h

-m '.tags[?]' .key termination_date -a .value -{1w}
    Same as the previous example, except this returns true if o['value'] will expire within
    the current week. In the real world, this example could be combined with the kind primary
    to filter out all EC2 instances whose termination_date tag will expire within the current
    week. The expression would look something like
        find aws/demo -k '*ec2*instance' -m '.tags[?]' .key termination_date -a .value -{1w}

-m '.tags[?]' .key \( sales -o product \)
    Returns true if m['tags'] has at least one object o s.t. o['key'] == sales OR product.
    In the real world, this example could be combined with the kind primary to filter out all
    EC2 instances that have a "sales" or "product" tag. The expression would look something
    like
        find aws/demo -k '*ec2*instance' -m '.tags[?]' .key \( sales -o product \)

-m .state.name pending -o running
    Returns true if m['state']['name'] == pending OR running. In the real world, this
    example could be combined with the kind primary to filter out pending/running EC2 instances.
    The expression would look something like
        find aws/demo -k '*ec2*instance' -m .state.name pending -o running

-m .vpcid vpc-0eb70f7f626d3db84
    Returns true if m['vpcid'] == vpc-0eb70f7f626d3db84. In the real world, this example
    could be combined with the kind primary to filter out EC2 instances attached to the VPC with ID
    vpc-0eb70f7f626d3db84. The expression would look something like
        find aws/demo -k '*ec2*instance' -m .vpcid vpc-0eb70f7f626d3db84

    NOTE: With regex/glob support, this example could be shortened to something like
    "-m .vpcid vpc-0eb.*"

-m '.mounts[?]' .type tmpfs
    Returns true if m['mounts'] has at least one object o s.t. o['type'] == tmpfs. In the
    real world, this example could be combined with the kind primary to filter out all Docker
    containers that have tmpfs mounts. The expression would look something like
        find docker -k '*container' -m '.mounts[?]' .type tmpfs
`
