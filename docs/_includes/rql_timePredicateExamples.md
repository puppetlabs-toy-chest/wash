```
["{{include.name}}", [">", "2020-01-01T22:15:52Z"]]
```

Returns true if the {{include.comparedThing}} is greater than `01/01/2020 10:15:52 PM UTC`. This example can also be expressed as

```
["{{include.name}}", [">", 1577916952]]
```

where `1577916952` is the time in UNIX seconds. Thus, time values can be specified as RFC3339 strings or as UNIX seconds.

The grammar lets you specify an NPE of time predicates so syntax like

```
["{{include.name}}", ["AND", [">", "2020-01-01T22:15:52Z"], ["<", "2020-02-01T00:00:00Z"]]]
```

```
["{{include.name}}", ["OR", [">", "2020-01-01T22:15:52Z"], ["<", "2020-02-01T00:00:00Z"]]]
```

```
["{{include.name}}", ["NOT", [">", "2020-01-01T22:15:52Z"]]]
```

works as you'd expect. Specifically,

* The first example returns true if the {{include.comparedThing}} is greater than `01/01/2020 10:15:52 PM UTC` AND less than `02/01/2020 00:00:00 AM UTC`.

* The second example returns true if the {{include.comparedThing}} is greater than `01/01/2020 10:15:52 PM UTC` OR less than `02/01/2020 00:00:00 AM UTC`.

* The third example returns true if the {{include.comparedThing}} is NOT greater than `01/01/2020 10:15:52 PM UTC`