---
title: News
---
{% for post in site.posts %}
  <article>
    <h3><a href="{{ post.url | remove: "/index.html" | relative_url }}">{{ post.title }}</a></h3>
    <small>Posted {{ post.date | date: "%m-%d-%y" }}</small>
  </article>
  {% endfor %}
