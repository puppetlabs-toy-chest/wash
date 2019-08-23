+++
title= "wash: the wide-area shell"
date= 2019-04-19T22:59:26-06:00
description = ""
draft= false
+++

Wash helps you manage your remote infrastructure using well-established UNIX-y patterns and tools to free you from having to remember multiple ways of doing the same thing.

<div id="horizontalmenu">
    • <a href="#introduction">introduction</a>
    • <a href="#getting-started">getting started</a>
    • <a href="#current-features">features</a>
    • <a href="#contributing">contributing</a>
    •
</div>

## Introduction

Have you ever had to:

* List all your AWS EC2 instances or Kubernetes pods?
* Read/cat a GCP Compute instance's console output, or an AWS S3 object's content?
* Exec a command on a Kubernetes pod or GCP Compute Instance?
* Find all AWS EC2 instances with a particular tag, or Docker containers/Kubernetes pods/GCP Compute instances with a specific label?

If so, then some parts of the following tables might look familiar to you. If not, then here's how AWS/Docker/Kubernetes/GCP recommends that you do some of these tasks.

<div style="width:200px">List all</div> | 
----------------------|--------------------------------------------------
AWS EC2 instances     | `aws ec2 describe-instances --profile foo --query 'Reservations[].Instances[].InstanceId' --output text`
Docker containers     | `docker ps --all`
Kubernetes pods       | `kubectl get pods --all-namespaces`
GCP Compute instances | `gcloud compute instances list`

<div style="width:200px">Read</div>         | 
--------------------------------------------|---
Console output of an EC2 instance           | `aws ec2 get-console-output --profile foo --instance-id bar`
Console output of a Google compute instance | `gcloud compute instances get-serial-port-output foo`
An S3 object's content                      | `aws s3api get-object content.txt --profile foo --bucket bar --key baz && cat content.txt && rm content.txt`
A GCP Storage object's content              | `gsutil cat gs://foo/bar`

<div style="width:200px">Exec `uname` on</div> | 
-----------------------------|---
An EC2 instance              | `ssh -i /path/my-key-pair.pem ec2-user@195.70.57.35 uname`
An a Docker container        | `docker exec foo uname`
Exec on a Kubernetes pod     | `kubectl exec foo uname`
On a Google Compute instance | `gcloud compute ssh foo --command uname`

<div style="width:200px">Find by 'owner' tag/label</div> | 
--------------------------|---
EC2 instances             | `aws ec2 describe-instances --profile foo --query 'Reservations[].Instances[].InstanceId' --filters Name=tag-key,Values=owner --output text`
Docker containers         | `docker ps --filter “label=owner”`
Kubernetes pods           | `kubectl get pods --all-namespaces --selector=owner`
Google Compute instance   | `gcloud compute instances list --filter=”labels.owner:*”`

That’s a lot of commands you have to use, applications you need to install, and DSLs you have to learn just to do some very fundamental and basic tasks. Now take a look at how you’d perform those same tasks with Wash:

<div style="width:200px">List all</div> | 
----------------------|---
AWS EC2 instances     | `find aws/foo -k '*ec2*instance'`
Docker containers     | `find docker -k '*container' `
Kubernetes pods       | `find kubernetes -k '*pod'`
GCP Compute instances | `find gcp -k '*compute*instance'`

<div style="width:200px">Read</div>         | 
--------------------------------------------|---
Console output of an EC2 instance           | `cat aws/foo/resources/ec2/instances/bar/console.out`
Console output of a Google compute instance | `cat gcp/<project>/compute/foo/console.out`
An S3 object's content                      | `cat aws/foo/resources/s3/bar/baz`
A GCP Storage object's content              | `cat gcp/<project>/storage/foo/bar`

<div style="width:200px">Exec `uname` on </div> | 
-----------------------------|---
An EC2 instance              | `wexec aws/foo/resources/ec2/instances/bar uname`
An a Docker container        | `wexec docker/containers/foo uname`
Exec on a Kubernetes pod     | `wexec kubernetes/<context>/<namespace>/pods/foo uname`
On a Google Compute instance | `wexec gcp/<project>/compute/foo uname`

<div style="width:200px">Find by 'owner' tag/label</div> | 
------------------------|---
EC2 instances           | `find aws/foo -k '*ec2*instance' -meta '.tags[?].key' owner`
Docker containers       | `find docker -k '*container' -meta '.labels.owner' -exists`
Kubernetes pods         | `find kubernetes -k '*pod' -meta '.metadata.labels.owner' -exists`
Google Compute instance | `find gcp -k '*compute*instance' -meta '.labels.owner' -exists`

From the table, we see that using Wash means:

* You no longer have to learn different commands to execute a task across different things. All you need is one command (`find` for List/Find; `cat` for Read; and `wexec` for Exec).

* You no longer have to install a bunch of different tools. All you need to install is the Wash binary.

* You no longer have to learn different DSLs for filtering stuff. All you need to learn is find's expression syntax and its individual primaries. Once you do that, you can filter on almost any conceivable property of your specific thing.

And this is only scratching the surface of Wash's capabilities. Checkout the screencast below to see some more (and to see Wash in action):

<script id="asciicast-mX8Mwa75rr1bJePLi3OnIOkJK" src="https://asciinema.org/a/mX8Mwa75rr1bJePLi3OnIOkJK.js" async></script>

## Getting started

Wash is distributed as a single binary, and the only prerequisite is [`libfuse`](https://github.com/libfuse/libfuse). Thus, getting going is pretty simple:

1. [Download](https://github.com/puppetlabs/wash/releases) the Wash binary for your platform
   * or install with `brew install puppetlabs/puppet/wash`
2. Install `libfuse`, if you haven't already
   * E.g. on MacOS using homebrew: `brew cask install osxfuse`
   * E.g. on CentOS: `yum install fuse fuse-libs`
   * E.g. on Ubuntu: `apt-get install fuse`
3. Start Wash
   * `./wash`


**NOTE:** Wash uses your system shell to provide the shell environment. It determines this using the `SHELL` environment variable or falls back to `/bin/sh`. See [wash shell](docs#wash-shell) on customizing your shell environment.

At this point, if you haven't already, you should start some resources that Wash can actually introspect. Otherwise, as Han Solo would say, "this is going to be a real short trip". So fire up some Docker containers, create some EC2 instances, toss some files into S3, launch a Kubernetes pod, etc. We've also provided a [tutorial](tutorial) that includes Docker and Kubernetes applications.

**NOTE:** Wash collects anonymous data about how you use it. See the [analytics docs](docs#analytics) for more details.

### Release announcements

You can watch for new releases of Wash on [Slack #announcements](https://puppetcommunity.slack.com/app_redirect?channel=announcements), the [puppet-announce](https://groups.google.com/forum/#!forum/puppet-announce) mailing list, or by subscribing to new releases on [GitHub](https://github.com/puppetlabs/wash).

### Known issues

#### On macOS

If using iTerm2, we recommend installing [iTerm2's shell integration](https://www.iterm2.com/documentation-shell-integration.html) to avoid [issue#84](https://github.com/puppetlabs/wash/issues/84).

If the `wash` daemon exits with a exit status of 255, that typically means that `wash` couldn't load the FUSE extensions. MacOS only allows for a certain (small) number of virtual devices on the system, and if all available slots are taken up by other programs then we won't be able to run. You can view loaded extensions with `kextstat`. More information in [this github issue for *FUSE for macOS*](https://github.com/osxfuse/osxfuse/issues/358).

## Current features

Wash does a lot already, with [more to come](https://github.com/puppetlabs/wash#roadmap):

* presents a filesystem hierarchy for all of your resources, letting you navigate them in normal, filesystem-y ways
* preserves history of all executed commands, facilitating debugging
* serves up an HTTP API for everything
* caches information, for better performance

We've implemented a number of handy Wash commands ([docs](docs#wash-commands)):

* `wash ls` - a version of `ls` that uses our API to enhance directory listings with Wash-specific info
  - _e.g. show you what primitives are supported for each resource_
* `wash meta` - emits a resource's metadata to standard out
* `wash exec` - uses the `exec` primitive to let you invoke commands against resources
* `wash find` - find resources using powerful selection predicates
* `wash tail -f` - follow updates to resources that support the `stream` primitive as well as normal files
* `wash ps` - lists running processes on indicated compute instances that support the `exec` primitive
* `wash history` - lists all activity through Wash; `wash history <id>` can be used to view logs for a specific activity
* `wash clear` - clears cached data for a sub-hierarchy rooted at the supplied path so Wash will re-request it

[Core plugins](docs#core-plugins) (and we're [adding more all the time](https://github.com/puppetlabs/wash#roadmap), see our [docs](docs#plugin-concepts) for how to help):

* [docker](docs#docker): containers and volumes
* [kubernetes](docs#kubernetes): pods, containers, and persistent volume claims
* [aws](docs#aws): EC2 and S3
* [gcp](docs#gcp): Compute Engine and Storage

[External plugins](docs/external_plugins):

* Wash allows for easy creation of out-of-process plugins using any language you want, from `bash` to `go` or anything in-between!
* Wash handles the plugin life-cycle. it invokes your plugin with a certain calling convention; all you have to do is supply the business logic
* users interact with external plugins the exact same way as core plugins; they are first-class citizens

Several external plugins have already been created:

* [Washhub](https://github.com/timidri/washhub) - navigate all your GitHub repositories at once
* [Washreads](https://github.com/MikaelSmith/washreads) - view your Goodreads bookshelves; also structured as an introduction to writing external plugins
* [Puppetwash](https://github.com/timidri/puppetwash) - view your Puppet (Enterprise) instances and information about the managed nodes
* [AWS IoT](https://gitlab.com/nwops/wash-iot) - view your AWS IoT devices and the shadow data from Wash

If you've created an external plugin, please put up a pull request to add it to this [list](https://github.com/puppetlabs/wash/edit/master/website/content/_index.md)!

For more information about future direction, see our [Roadmap](https://github.com/puppetlabs/wash#roadmap)!

## Contributing

There are tons of ways to get involved with Wash, whether or not you're a programmer!

- Come and hang out with us on [Slack](https://puppetcommunity.slack.com/app_redirect?channel=wash)! Feel free to ask questions, answer questions from other folks, or just lurk. Come and talk to us about things about modern infra you find [complex or infuriating](https://landscape.cncf.io/), or what your [favorite hacking movie scenes](https://www.youtube.com/watch?v=u1Ds9CeG-VY) are, or the [best monospaced font](https://fonts.google.com/specimen/Inconsolata). 

- Have something cool that you'd like connect up to Wash? We'd love to hear your ideas, and help you figure out how to do it! We don't care if you want Wash to talk to a network device, a smart lightbulb, your bluetooth-enabled espresso scale, or just more types of resources from cloud providers. 

- Are you an artist? Design some Wash-related artwork or a logo, and we'll see about putting it into the rotation for the site!

- Are you an old skool command-line gearhead with, like, *opinions* about how things should work on a command line? We'd love your help figuring out how the shell experience for Wash should work. How can our unixy Wash commands behave better? Are there new commands we should build? What colors and formatting should we use for `wash ls`? If we implemented `wash fortune`, what quotes should be in there?!

- Did you script something cool that uses Wash under the hood? Please let us know, and how we can help!

- Can you sling HTML, or Markdown? This site is built using Hugo, and the source is in our [github repo](https://github.com/puppetlabs/wash/tree/master/website). We'd love help documenting stuff!

- Did you give a talk or presentation on Wash? Give us the link, so we can help promote it!

Come check us out on [github](https://github.com/puppetlabs/wash), and in particular check out the [contribution guidelines](https://github.com/puppetlabs/wash/blob/master/CONTRIBUTING.md) and the [code of conduct](https://github.com/puppetlabs/wash/blob/master/CODE_OF_CONDUCT.md).
