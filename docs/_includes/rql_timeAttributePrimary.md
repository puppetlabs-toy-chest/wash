### {{include.name}}

The `{{include.name}}` primary constructs a predicate on an entry's `{{include.name}}` attribute.

#### Examples

{% assign comparedThing = "entry's " | append: include.name | append: ' attribute' %}
{% include rql_timePredicateExamples.md name=include.name comparedThing=comparedThing %}