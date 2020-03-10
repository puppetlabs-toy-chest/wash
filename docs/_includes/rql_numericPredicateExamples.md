```
["{{include.name}}", [">", 10]]
```

Returns true if the {{include.comparedThing}} is greater than 10{{include.units}}. This example can also be expressed as

```
["{{include.name}}", [">", "10"]]
```

where `"10"` is the stringified numeric value. Thus, numeric values can be specified as JSON numbers or strings. The string representation's useful for large numbers.

The grammar lets you specify an NPE of Numeric predicates so syntax like

```
["{{include.name}}", ["AND", [">", 10], ["<", 20]]]
```

```
["{{include.name}}", ["OR", [">", 10], ["<", 20]]]
```

```
["{{include.name}}", ["NOT", [">", 10]]]
```

works as you'd expect. Specifically,

* The first example returns true if the {{include.comparedThing}} is greater than 10{{include.units}} AND less than 20{{include.units}}.

* The second example returns true if the {{include.comparedThing}} is greater than 10{{include.units}} OR less than 20{{include.units}}.

* The third example returns true if the {{include.comparedThing}} is NOT greater than 10{{include.units}}.