---
title: <b>Wash</b>
---

<div class="flex-row">
  <p class="intro-normal">A UNIX-like shell for managing your cloud native and non-cloud native things.</p>
  <div class="flex-column pad-left">
    <a class="intro-normal center btn" href="{{ '/getting_started' | relative_url }}">GET&nbsp;STARTED</a>
    <!-- update css and javascript -->
    <a class="intro-normal center version">Last release: unknown</a>
  </div>
</div>
<script type="text/javascript">
$.get("https://api.github.com/repos/puppetlabs/wash/releases/latest", function(data) {
  $('.version')
    .text("Last release: " + $.timeago(data.published_at))
    .attr("href", "https://github.com/puppetlabs/wash/releases/tag/" + data.tag_name)
    .attr("title", "Version " + data.tag_name)
});
</script>

<p class="intro-large">With Wash</p>

* <p class="intro-normal">You can use <code>ls</code> to list, <code>cat</code> to read, and <code>wexec</code> to run commands on all your things. No more switching between confusing CLI tools.</p>
* <p class="intro-normal">You can use <code>find</code> to filter anything on anything. No more complicated query DSLs.</p>
* <p class="intro-normal">You can <code>cd</code> through a vendor's API. No more navigating complex console UIs.</p>

<p class="intro-large">See for yourself</p>

<!-- Display the demos -->
# **AWS**
<div id="aws-demo" >
  {% capture aws_annotation %}
  The EC2 instance <code>find</code> query shown above (<code>find . -k '*instance' -m '.state.name' running -m '.tags[?].key' owner</code>) returns all running EC2 instances with the 'owner' tag.
  {% endcapture %}

  {% include screencast.html name="intro/aws" poster="0:17" annotation=aws_annotation %}
</div>

# **GCP**
<div id="gcp-demo" >
  {% capture gcp_annotation %}
  The compute instance <code>find</code> query shown above (<code>find . -k '*instance' -m '.status' RUNNING -m '.labels.owner' -exists</code>) returns all running compute instances with the 'owner' label.
  {% endcapture %}

  {% include screencast.html name="intro/gcp" poster="0:18" annotation=gcp_annotation %}
</div>

# **Kubernetes**
<div id="kubernetes-demo" >
  {% capture kubernetes_annotation %}
  The pods <code>find</code> query shown above (<code>find . -k '*pod' -m '.status.phase' Running -m '.metadata.labels.pod-template-hash' -exists</code>) returns all running pods with the 'pod-template-hash' label.
  {% endcapture %}

  {% include screencast.html name="intro/kubernetes" poster="0:18" annotation=kubernetes_annotation %}
</div>

# **Docker**
<div id="docker-demo" >
  {% capture docker_annotation %}
  The container <code>find</code> query shown above (<code>find . -k '*container' -m '.state' running -m '.labels.com\.docker\.compose\.version' -exists</code>) returns all running containers with the 'com.docker.compose.version' label.
  {% endcapture %}

  {% include screencast.html name="intro/docker" poster="0:18" annotation=docker_annotation %}
</div>

# **Other**
<div id="external-plugin-demo" >
  {% capture external_plugin_annotation %}
  The Spotify plugin shows off Wash's greatest power: its ability to talk to <i>anything</i> via the external plugin interface. And when we say anything, we really do mean anything. We mean other cloud native vendors like OpenStack or Azure. We mean personal IoT devices like network devices, smart lightbulbs, or bluetooth-enabled espresso scales. We mean IT infrastructure like Puppet nodes or Bolt inventory files. And we also mean some truly bizarre APIs like Goodreads or Fandango. Thus if you've got some other things you'd like to <code>cd</code> and <code>ls</code> through, filter with <code>find</code>, read with <code>cat</code>, or <a href="{{ '/docs#actions' | relative_url }}">more</a>, then give Wash a try. We already have some <a href="{{ '/docs/external-plugins#example-plugins' | relative_url }}">community-built external plugins</a> that you can use. If those aren't enough, then you can write your own external plugin in <i>any</i> language you like (think Bash, Ruby, Python, Go). The sky is the limit.
  {% endcapture %}

  {% include screencast.html name="intro/external-plugins" poster="0:15" annotation=external_plugin_annotation %}
</div>
