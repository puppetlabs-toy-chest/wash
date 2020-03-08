---
title: Resource Query Language
---

* [Background](#background)
* [AST Grammar](#ast-grammar)
* [Primaries](#primaries)
  * [action](#action)
  * [boolean](#boolean)
  * [name](#name)
  * [cname](#cname)
  * [path](#path)
  * [kind](#kind)
  * [atime](#atime)
  * [crtime](#crtime)
  * [ctime](#ctime)
  * [mtime](#mtime)
  * [size](#size)
  * [meta](#meta)
* [Detailed Meta Primary Overview](#detailed-meta-primary-overview)

## Background

The resource query language (RQL) lets you query Wash resources. You can submit your queries to the API's `find` endpoint. Here's an example query

```
Eniss-MacBook-Pro:wash enisinan$ curl -X POST --unix-socket /tmp/WASH_SOCKET --header "Content-Type: application/json" --data '["kind", ["glob", "*ec2*instance"]]' 'http://localhost:/fs/find?path=/tmp/WASH_MOUNT/aws/wash' 2>/dev/null | jq
[
  {
    "type_id": "aws::github.com/puppetlabs/wash/plugin/aws/ec2Instance",
    "path": "/tmp/WASH_MOUNT/aws/wash/resources/ec2/instances/i-04621c13583930e6c",
...
```

> Note that I started my own Wash server instance with `WASH_SOCKET="/tmp/WASH_SOCKET" ./wash server /tmp/WASH_MOUNT`

This query returns all entries under the `aws/wash` entry whose `kind` matches the glob `*ec2*instance`. Informally, this query returns all AWS EC2 instances under the `wash` profile.

You can view the [API docs]({{'/docs/api' | relative_url}}) for more details on the `find` endpoint, including its query parameters (not to be confused with an RQL query, which is specified in the request body).

## AST Grammar

This section documents the RQL's AST grammar. For convenience, let `PE <PredicateType>` denote the following grammar (where `PE` => `PredicateExpression`).

```
PE <PredicateType> :=
  [BinaryOp, PE <PredicateType>, PE <PredicateType>] |
  <PredicateType>

BinaryOp := "AND" | "OR"
```

where `<PredicateType>` has its own AST grammar. Informally, `PE <PredicateType>` translates to `a predicate expression of <PredicateType>s`. For example, `PE Primary` translates to `a predicate expression of primaries`.

Similarly, let `NPE <PredicateType>` denote the following grammar (where `NPE` => `NegatablePredicateExpression`):

```
NPE <PredicateType> :=
  [UnaryOp,  NPE <PredicateType>]                       |
  [BinaryOp, NPE <PredicateType>, NPE <PredicateType>]  |
  <PredicateType>

UnaryOp := ‘NOT’
```

Then the RQL AST can be expressed as

```
Query := PE Primary

Primary :=
  Action  |
  Boolean |
  Name    |
  CName   |
  Path    |
  Kind    |
  Atime   |
  Crtime  |
  Ctime   |
  Mtime   |
  Size    |
  Meta

Action  := [“action”, NPE ActionPredicate]
Boolean := BooleanPredicate
Name    := [“name”,   NPE StringPredicate]
CName   := [“cname”,  NPE StringPredicate]
Path    := [“path”,   NPE StringPredicate]
Kind    := [“kind”,   NPE StringPredicate]
Atime   := [“atime”,  NPE TimePredicate]
Crtime  := [“crtime”, NPE TimePredicate]
Ctime   := [“ctime”,  NPE TimePredicate]
Mtime   := [“mtime”,  NPE TimePredicate]
Size    := SizePredicate
Meta    := [“meta”,   PE ObjectPredicate]

ActionPredicate := 
  "list"   |
  "read"   |
  "write"  |
  "stream" |
  "exec"   |
  "delete"

ValuePredicate :=
  ObjectPredicate                    |
  ArrayPredicate                     |
  NullPredicate                      |
  BooleanPredicate                   |
  [“number”, NPE NumericPredicate]   |
  [“time”,   NPE TimePredicate]      |
  [“string”, NPE StringPredicate]

ObjectPredicate        := [“object”, SizePredicate | ObjectElementPredicate]
ObjectElementPredicate := [ObjectElementSelector, NPE ValuePredicate]
ObjectElementSelector  :=  [“key”, [“=”, <string>]]

ArrayPredicate        := [“array”, SizePredicate | ArrayElementPredicate]
ArrayElementPredicate := [ArrayElementSelector, NPE ValuePredicate]]
ArrayElementSelector  := “some” | “all”  | <array_index>

SizePredicate := [“size”, NPE NumericPredicate (n >= 0)]

NullPredicate := null

BooleanPredicate := true | false

NumericPredicate  := [ComparisonOp, Number]
ComparisonOp      := “<” | “>” | “<=” | “>=” | “=” | “!=”

StringPredicate :=
  [“glob”,   <glob>]   |
  [“regex”,  <regex>]  |
  [“=”,      <string>]

TimePredicate := [ComparisonOp, TimeValue]
```

This grammar tells us several things:

* An RQL query is a predicate expression of primaries. A primary is a predicate on a specific field of an entry. Examples of primaries include `name`, `cname`, and `ctime`. See the [primaries](#primaries) section for all of the available primaries.

* Primaries take predicates, specifically predicate expressions.

## Primaries

### action

The `action` primary constructs a predicate on an entry's supported actions.

#### Examples

```
["action", "exec"]
```

Returns true if the entry supports the `exec` action.

### boolean

The `boolean` primary returns true or false depending on the predicate's value.

#### Examples

```
true
```

Returns true.

### name

The `name` primary constructs a predicate on the entry's name.

#### Examples

{% include rql_stringPredicateExamples.md name="name" comparedThing="entry's name" %}

### cname

The `cname` primary constructs a predicate on the entry's cname.

#### Examples

{% include rql_stringPredicateExamples.md name="cname" comparedThing="entry's cname" %}

### path

The `path` primary constructs a predicate on the entry's path _relative_ to the start entry's path. For example, if the starting entry is `foo` with children `bar` and `baz`, then `bar` and `baz`'s path is `bar` and `baz`, respectively. If `qux` is a child of `bar`, then its path is `bar/qux`. Similarly if `quuz` is a child of `baz`, then its path is `baz/quuz`.

#### Examples

{% include rql_stringPredicateExamples.md name="path" comparedThing="entry's path" %}

### kind

The `kind` primary constructs a predicate on the entry's kind. The entry's kind is its schema path but without the `<start_entry_label>`. An entry's schema path is constructed as `<start_entry_label>/<parent1_label>/.../<label>`,
where `<start_entry_label>` is the label of the start entry (the stree root). Thus, an entry's kind is `<parent1_label>/.../<label>`.

You can use `stree <start_entry_path>` to determine an entry's kind. For example, if `<start_entry_path>` is `docker`, then

```
wash . ❯ stree docker
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
```

We see from the output that a Docker container's kind is `containers/container`. Similarly, a Docker volume directory's kind is `volumes/volume/dir` OR `volumes/volume/dir/dir`. The latter comes from the definition -- a volume directory has more than one possible schema path and hence more than one possible kind.

**Note:** The `kind` primary will always return false if the plugin doesn't support entry schemas.

#### Examples

{% include rql_stringPredicateExamples.md name="kind" comparedThing="entry's kind" %}

{% include rql_timeAttributePrimary.md name="atime" %}

{% include rql_timeAttributePrimary.md name="crtime" %}

{% include rql_timeAttributePrimary.md name="ctime" %}

{% include rql_timeAttributePrimary.md name="mtime" %}

### size

The `size` primary constructs a predicate on the entry's size attribute. Note that all numeric values should be unsigned integers (>= 0); otherwise, the `find` endpoint will return an error.

#### Examples

{% include rql_numericPredicateExamples.md name="size" comparedThing="entry's size attribute" units=" bytes" %}

### meta

The `meta` primary constructs a predicate on the entry's metadata. If the `fullmeta` option is not set, then this will be the entry's _partial_ metadata. Otherwise if `fullmeta` is true, then it will be the entry's _full_ metadata.

#### Examples

```
["meta", ["object", [["key", "foo"], ["number", ["=", 5]]]]]
```

Returns true if `m['foo'] == 5`, where `m` is the entry's metadata.

```
["meta",
  ["AND",
    ["object", [["key", "foo"], ["number", ["=", 5]]]],
    ["object", [["key", "bar"], ["string", ["=", "baz"]]]]]]
```

Returns true if `m['foo'] == 5` AND `m['bar'] == "baz"`, where `m` is the entry's metadata.

```
["meta",
  ["OR",
    ["object", [["key", "foo"], ["number", ["=", 5]]]],
    ["object", [["key", "bar"], ["string", ["=", "baz"]]]]]]
```

Returns true if `m['foo'] == 5` OR `m['bar'] == "baz"`, where `m` is the entry's metadata.

**Note:** The `meta` primary takes PE ObjectPredicate, _not_ NPE ObjectPredicate. Thus something like

```
["meta", ["NOT", ["object", [["key", "foo"], ["number", ["=", 5]]]]]]
```

is an invalid query.

The following example is more "real-worldly".

```
["meta",
  ["object",
    [["key", "tags"],
    ["array",
      ["some",
        ["AND",
          ["object",
            [["key", "key"),
              ["string", ["=", "termination_date"]]]]],
          ["object",
            [["key", "value"),
              ["time", ["<", "2017-08-07T13:55:25.680464+00:00"]]]]]]]]]]]
```

Returns true if `m['tags']` has at least one object o s.t. `o['key'] == termination_date` AND `o['value'] < 8/07/2017 ...` (i.e. `o['value']` has expired). In the real world, this example could be combined with the `kind` primary to filter out all EC2 instances whose `termination_date` tag expired. The full request to `find` would look something like

```
curl -X POST --unix-socket /tmp/WASH_SOCKET --header "Content-Type: application/json" --data @rql_query.json 'http://localhost:/fs/find?path=/tmp/WASH_MOUNT/aws/wash' 2>/dev/null | jq
```

where `rql_query.json` looks something like

```
["AND",
  ["kind", ["glob", "*ec2*instance"]],
  ["meta",
    ["object",
      [["key", "tags"],
      ["array",
        ["some",
          ["AND",
            ["object",
              [["key", "key"),
                ["string", ["=", "termination_date"]]]]],
            ["object",
              [["key", "value"),
                ["time", ["<", "2017-08-07T13:55:25.680464+00:00"]]]]]]]]]]]]
```

## Detailed Meta Primary Overview

### Object Predicate

A predicate on a JSON object can either be a predicate on the object's size or on its elements.

#### Size Predicate

{% include rql_numericPredicateExamples.md name="object" comparedThing="object's size" units=" elements" %}

#### Element Predicate

An element predicate would look something like

```
["object", [["key", "foo"], ["number", ["=", 5]]]]
```

This returns true if `o['foo'] == 5`. Note that the `meta` primary will find the first key that matches `foo`. This is the first key such that `upcase(key) == FOO`. So, the above predicate would return true if `o['foo'] == 5`, or if `o['fOo'] == 5`, or if `o['FOO'] == 5` depending on which key exists. If there is no matching key, then the predicate will return false. If there are multiple matching keys, then the `meta` primary will choose one at random.

An element predicate can also take an NPE of value predicates. Below are some examples of this.

{% include rql_valuePredicateExamples.md %}

### Array Predicate

A predicate on a JSON array can either be a predicate on the array's size or on its elements.

#### Size Predicate

{% include rql_numericPredicateExamples.md name="array" comparedThing="array's size" units=" elements" %}

#### Element Predicate

An element predicate could look something like

```
["array", ["some", ["number", ["=", 5]]]]
```

```
["array", ["all", ["number", ["=", 5]]]]
```

```
["array", [1, ["number", ["=", 5]]]]
```

The first example returns true if `a[?] == 5` where `a[?]` means "some element in the array". The second example returns true if `a[*] == 5`, where `a[*]` means "all elements in the array". Finally, the third example returns true if `a[1] == 5`, where `a[1]` means the element at index 1. The last example is an instance of the more general `["array", [n, p]]` example, which returns true if `a[n]` satisfies p, where `n` is some array index value.

An element predicate can also take an NPE of value predicates. Below are some examples of this.

{% include rql_valuePredicateExamples.md isArray=true %}

### Null Predicate

```
null
```

This returns true if the value's `null`, false otherwise.

### Boolean Predicate

```
true
```

Returns true if the value's a Boolean value that's equal to `true`.

```
false
```

Returns true if the value's a Boolean value that's equal to `false`.

### Numeric Predicate

{% include rql_numericPredicateExamples.md name="number" comparedThing="numeric value" %}

**Note:** A numeric predicate will always return false for non-Numeric values like e.g. Boolean values.

### Time Predicate

{% include rql_timePredicateExamples.md name="time" comparedThing="time value" %}

**Note:** A time predicate will always return false for non-Time values like e.g. Boolean values.

### String Predicate

{% include rql_stringPredicateExamples.md name="string" comparedThing="string value" %}

**Note:** A string predicate will always return false for non-String values like e.g. Boolean values.

### Subtleties

Consider a value expression like

```
["NOT", ["number", [">", 5]]]
```

Then this will return true for _all_ numeric values `<= 5`. It will also return true for all _non-numeric values_ since `["number", [">", 5]]` returns false for those. If your intent is to negate the `[">", 5]` part but still return false for non-numeric values, then you should move the `NOT` inside the numeric predicate. So the value expression should be written as

```
["number", ["NOT", [">", 5]]]
```

Similar subtleties exist for String and Time predicates. In general, for any predicates that are of the form `[<value_type>, p]`, if your intent is to negate `p` while still returning false for mis-typed values, then you should write the negation as `[<value_type>, [NOT, p]]` instead of `[NOT, [<value_type>, p]]`.

Finally, something like

```
["NOT", true]
```

will return false for `true` Boolean values, but true for all _non-Boolean values_ since `true` returns false for those. If your intent is to negate the `true` part, then just pass-in `false` instead. So you should use

```
false
```

to clarify this intent.

So if there are subtleties with `NOT`, then why do we allow NPE ValuePredicates? Why not limit things to just PE ValuePredicates? The reason is because `NOT` makes it easier to express other predicates without adding-in extra RQL syntax. For example, something like

```
["object", [["key", "foo"], ["OR", null, ["NOT", null]]]]
```

can be used to test that `o['foo']` exists. Something like

```
["NOT", ["object", [">=", 0]]]
```

can be used to test for a non-Object value (since all object values have `>= 0` size).







