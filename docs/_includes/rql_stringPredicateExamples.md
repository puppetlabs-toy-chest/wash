```
["{{include.name}}", ["glob", "foo"]]
```

Returns true if the {{include.comparedThing}} matches the glob `foo`.

```
["{{include.name}}", ["regex", "foo"]]
```

Returns true if the {{include.comparedThing}} matches the regex `foo`.

```
["{{include.name}}", ["=", "foo"]]
```

Returns true if the {{include.comparedThing}} equals `foo`.

The grammar lets you specify an NPE of String predicates so syntax like

```
["{{include.name}}", ["AND", ["glob", "foo"], ["regex", "bar"]]]
```

```
["{{include.name}}", ["OR", ["glob", "foo"], ["regex", "bar"]]]
```

```
["{{include.name}}", ["NOT", ["glob", "foo"]]]
```

work as you'd expect. Specifically,

* The first example returns true if the {{include.comparedThing}} matches the glob `foo` AND matches the regex `bar`.

* The second example returns true if the {{include.comparedThing}} matches the glob `foo` OR matches the regex `bar`.

* The third example returns true if the {{include.comparedThing}} does NOT match the glob `foo`.