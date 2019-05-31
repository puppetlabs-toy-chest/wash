+++
title= "wash: the wide-area shell"
date= 2019-04-19T22:59:26-06:00
description = ""
draft= false
+++

Wash helps you deal with all your remote or cloud-native infrastructure using the UNIX-y patterns and tools you already know and love.

<div id="horizontalmenu">
    • <a href="#introduction">introduction</a>
    • <a href="#current-features">features</a>
    • <a href="#getting-started">installation</a>
    • <a href="#contributing">contributing</a>
    •
</div>

<script id="asciicast-248077" src="https://asciinema.org/a/248077.js" async></script>

## Introduction

Wash aims to:

* make navigating stuff like servers, containers, or APIs as easy as navigating a local filesystem
* make scripting across your new-fangled infrastructure as easy as writing a local shell script
* render into text that which can be rendered into text (cuz text is a universal interface!) for easy viewing, editing, and UNIXy slicing-and-dicing
* build equivalents of basic, UNIX tools to support the above goals (but reuse existing ones if they work!)
* let you easily extend the system in whatever language you want
* be extremely simple to get up-and-running; if it takes you more than a few minutes, let us know!

Exploring, understanding, and inspecting modern infrastructure should be simple and straightforward. Whether it's containers, VMs, network devices, IoT stuff, or anything in between...they all have different ways of enumerating what you have, getting a stream of output, running commands, etc. Every vendor has its own tools and APIs that expose these features, each one different, each one bespoke. Thus, they are difficult to compose together to solve higher-level problems. And that's no fun at all!

[UNIX's philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Origin) and abstractions have worked for decades. They're pretty good, and more importantly, they're _familiar_ to millions of people. Wash intends to apply those same philosophies and abstractions to modern, distributed infrastructure.

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

At this point, if you haven't already, you should fire up some resources that Wash can actually introspect. Otherwise, as Han Solo would say, "this is going to be a real short trip". So fire up some Docker containers, create some EC2 instances, toss some files into S3, launch a Kubernetes pod, etc. 

For more of a guided tour that includes spinning up some resources Wash can talk to, check out our [`docker compose` example](https://github.com/puppetlabs/wash#wash-by-example).

Once the server is up, you can use vanilla `ls`, `cd`, etc. to explore. You can then start experimenting with Wash subcommands, like `wash ls` and `wash tail`, to navigate that filesystem in a more Wash-optimized way. Wash provides wrappers for some of these; you can usually find the native POSIX variants in `/usr/bin` or `/bin`.

When you're done, `exit` the shell.

## Current features

Wash does a lot already, with [more to come](https://github.com/puppetlabs/wash#roadmap):

* presents a filesystem hierarchy for all of your resources, letting you navigate them in normal, filesystem-y ways
* preserves history of all executed commands, facilitating debugging
* serves up an HTTP API for everything
* caches information, for better performance

We've implemented a number of handy Wash subcommands ([docs](/wash/docs#wash-commands)):

* `wash ls` - a version of `ls` that uses our API to enhance directory listings with Wash-specific info
  - _e.g. show you what primitives are supported for each resource_
* `wash meta` - emits a resource's metadata to standard out
* `wash exec` - uses the `exec` primitive to let you invoke commands against resources
* `wash find` - find resources using powerful selection predicates
* `wash tail -f` - follow updates to resources that support the `stream` primitive as well as normal files
* `wash ps` - lists running processes on indicated compute instances that support the `exec` primitive
* `wash history` - lists all activity through Wash; `wash history <id>` can be used to view logs for a specific activity
* `wash clear` - clears cached data for a subhierarchy rooted at the supplied path so Wash will re-request it

[Core plugins](/wash/docs#core-plugins) (and we're [adding more all the time](https://github.com/puppetlabs/wash#roadmap), see our [docs](/wash/docs#plugin-concepts) for how to help):

* [docker](/wash/docs#docker): containers and volumes
* [kubernetes](/wash/docs#kubernetes): pods, containers, and persistent volume claims
* [aws](/wash/docs#aws): EC2 and S3

[External plugins](/wash/docs/external_plugins):

* Wash allows for easy creation of out-of-process plugins using any language you want, from `bash` to `go` or anything in-between!
* Wash handles the plugin lifecycle. it invokes your plugin with a certain calling convention; all you have to do is supply the business logic
* users interact with external plugins the exact same way as core plugins; they are first-class citizens

For more information about future direction, see our [Roadmap](https://github.com/puppetlabs/wash#roadmap)!

## Contributing

There are tons of ways to get involved with Wash, whether or not you're a programmer!

- Come and hang out with us on [Slack](https://puppetcommunity.slack.com/app_redirect?channel=wash)! Feel free to ask questions, answer questions from other folks, or just lurk. Come and talk to us about things about modern infra you find [complex or infuriating](https://landscape.cncf.io/), or what your [favorite hacking movie scenes](https://www.youtube.com/watch?v=u1Ds9CeG-VY) are, or the [best monospaced font](https://fonts.google.com/specimen/Inconsolata). 

- Have something cool that you'd like connect up to Wash? We'd love to hear your ideas, and help you figure out how to do it! We don't care if you want Wash to talk to a network device, a smart lightbulb, your bluetooth-enabled espresso scale, or just more types of resources from cloud providers. 

- Are you an artist? Design some Wash-related artwork or a logo, and we'll see about putting it into the rotation for the site!

- Are you an old skool command-line gearhead with, like, *opinions* about how things should work on a command line? We'd love your help figuring out how the shell experience for Wash should work. How can our unixy Wash subcommands behave better? Are there new subcommands we should build? What colors and formatting should we use for `wash ls`? If we implemented `wash fortune`, what quotes should be in there?!

- Did you script something cool that usees Wash under the hood? Please let us know, and how we can help!

- Can you sling HTML, or Markdown? This site is built using Hugo, and the source is in our [github repo](https://github.com/puppetlabs/wash/tree/master/website). We'd love help documenting stuff!

- Did you give a talk or presentation on Wash? Give us the link, so we can help promote it!

Come check us out on [github](https://github.com/puppetlabs/wash), and in particular check out the [contribution guidelines](https://github.com/puppetlabs/wash/blob/master/CONTRIBUTING.md) and the [code of conduct](https://github.com/puppetlabs/wash/blob/master/CODE_OF_CONDUCT.md).
