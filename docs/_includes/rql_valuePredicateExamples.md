{% if include.isArray %}
  {% assign ctype="array" %}
  {% assign selector='some' %}
  {% assign svp="a[?]" %}
{% else %}
  {% assign ctype="object" %}
  {% assign selector='["key", "foo"]' %}
  {% assign svp="o['foo']" %}
{% endif %}

```
["{{ctype}}", [{{selector}}, ["object", [["key", "bar"], ["number", ["=", 5]]]]]]
```

Returns true if `{{svp}}['bar'] == 5`.{% if include.isArray %} Note that this is read as _return true if some element in the array is an object `o` s.t. `o['bar'] == 5`_.{% endif %}

```
["{{ctype}}", [{{selector}}, ["array", ["some", ["number", ["=", 5]]]]]]
```

Returns true if `{{svp}}[?] == 5`.{% if include.isArray %} Note that this is read as _return true if some element in the array is an array `b` s.t. `b[?] == 5`_ (some element in `b` is `5`).{% else %} Note that this is read as _return true if `{{svp}}` is an array such that some element in the array is a numeric value that is equal to 5_.{% endif %}

```
["{{ctype}}", [{{selector}}, null]]
```

Returns true if `{{svp}} == null`.{% if include.isArray %} Note that this is read as _return true if some element in the array is null_.{% endif %}

```
["{{ctype}}", [{{selector}}, true]]
```

Returns true if `{{svp}} == true`.{% unless include.isArray %} Note that this will return false if `{{svp}}` is a non-Boolean value.{% endunless %}

```
["{{ctype}}", [{{selector}}, ["number", [">", 5]]]]
```

Returns true if `{{svp}} > 5`.{% unless include.isArray %} Note that this will return false if `{{svp}}` is a non-Numeric value.{% endunless %}

```
["{{ctype}}", [{{selector}}, ["string", ["glob", "foo"]]]]
```

Returns true if `{{svp}}` matches the glob `foo`.{% unless include.isArray %} Note that this will return false if `{{svp}}` is not a String value.{% endunless %}

```
["{{ctype}}", [{{selector}}, ["time", [>, "2020-01-01T22:15:52Z"]]]]
```

Returns true if `{{svp}} > 01/01/2020 ...`.{% unless include.isArray %}Note that this will return false if `{{svp}}` is not a Time value.{% endunless %}

The grammar lets you specify an NPE of value predicates so syntax like

```
["{{ctype}}", [{{selector}}, ["AND", ["number", ["<", 10]], true]]]
```

```
["{{ctype}}", [{{selector}}, ["OR", ["number", ["<", 10]], ["string", ["glob", "foo"]]]]]
```

```
["{{ctype}}", [{{selector}}, ["NOT", ["time", [">", "2020-01-01T22:15:52Z"]]]]]
```

works as you'd expect. Specifically,

* The first example returns true if `{{svp}} < 10 AND {{svp}} == true`. Since it is impossible for {% if include.isArray %}any element in the array{% else %}`{{svp}}`{% endif %} to be both a Numeric and a Boolean value, this example will always return false.

* The second example returns true if `{{svp}} < 10 OR {{svp}} =~ glob 'foo'` where `=~` is read as "matches".

* The third example returns true if `! {{svp}} > 01/01/2020 ...`.
