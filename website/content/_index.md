+++
title= "wash landing page"
date= 2019-04-19T22:59:26-06:00
description = ""
draft= false
+++

`wash` helps you deal with all your remote or cloud-native infrastructure using the UNIX-y patterns and tools you already know and love.

_TBD: Screencast_

`wash` aims to:

* make navigating stuff like servers, containers, or APIs as easy as navigating a local filesystem
* make scripting across your new-fangled infrastructure as easy as writing a local shell script
* render into text that which can be rendered into text (cuz text is a universal interface!) for easy viewing, editing, and UNIXy slicing-and-dicing
* build equivalents of basic, UNIX tools to support the above goals (but reuse existing ones if they work!)
* let you easily extend the system in whatever language you want

## Getting started

`wash` is distributed as a single binary, and the only prerequisite is [`libfuse`](https://github.com/libfuse/libfuse). Thus, getting going is pretty simple:

1. [Download](https://github.com/puppetlabs/wash/releases) the `wash` binary for your platform
2. Install `libfuse`, if you haven't already
   * E.g. on MacOS using homebrew: `brew cask install osxfuse`
   * E.g. on CentOS: `yum install fuse fuse-libs`
   * E.g. on Ubuntu: `apt-get install fuse`
3. Start the server
   * `./wash server wash-root-dir`

That should do it! Once the server is up, you can use vanilla `ls`, `cd`, etc. to explore `wash-root-dir`. You can then start experimenting with `wash` subcommands, like `wash ls` and `wash tail`, to navigate that filesystem in a more `wash`-optimized way.

For more of a guided tour, check out our [`docker compose` example](https://github.com/puppetlabs/wash#wash-by-example).

When you're done, make sure there are no processes still using `wash-root-dir`, and you can just `Ctrl-C` the server.

## Motivation

Exploring, understanding, and inspecting modern infrastructure should be simple and straightforward. Whether it's containers, VMs, network devices, IoT stuff, or anything in between...they all have different ways of enumerating what you have, getting a stream of output, running commands, etc. Every vendor has its own tools and APIs that expose these features, each one different, each one bespoke. Thus, they are difficult to compose together to solve higher-level problems. And that's no fun at all!

[UNIX's philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Origin) and abstractions have worked for decades. They're pretty good, and more importantly, they're _familiar_ to millions of people. `wash` intends to apply those same philosophies and abstractions to modern, distributed infrastructure.


## Features

`wash` can already do quite a bit (listed below), and we've got [even more planned](https://github.com/puppetlabs/wash#roadmap)!

The `wash` daemon:

* presents a FUSE filesystem hierarchy for all of your resources, letting you navigate them in normal, filesystem-y ways
* preserves history of all executed commands, facilitating debugging
* serves up an HTTP API for everything
* caches information, for better performance

`wash` supports the following primitives for resources it knows about:

* `list` - lets you ask any resource what's contained inside of it, and what primitives it supports. 
  - _e.g. listing a Kubernetes pod returns its constituent containers_
* `read` - lets you read the contents of a given resource
  - _e.g. represent an EC2 instance's console output as a regular file you can open in a regular editor_
* `stream` - gives you streaming-read access to a resource
  - _e.g. to let you follow a container's output as its running_
* `exec` - lets you execute a command against a resource
  - _e.g. run a shell command inside a container, or on an EC2 vm, or on a routerOS device, etc._

We've implemented a number of handy `wash` subcommands:

* `wash ls` - a version of `ls` that uses our API to enhance directory listings with `wash`-specific info
  - _e.g. show you what primitives are supported for each resource_
* `wash meta` - emits a resource's metadata to standard out
* `wash exec` - uses the `exec` primitive to let you invoke commands against resources
* `wash find` - find resources using powerful selection predicates (WIP)
* `wash tail -f` - follow updates to resources that support the `stream` primitive as well as normal files
* `wash ps` - lists running processes on indicated compute instances that support the `exec` primitive
* `wash history` - lists all activity through `wash`; `wash history <id>` can be used to view logs for a specific activity
* `wash clear` - clears cached data for a subhierarchy rooted at the supplied path so `wash` will re-request it

Core plugins (see the _Roadmap_ below for more details)

* `docker` - presents a filesystem hierarchy of containers and volumes
  - found from the local socket or via `DOCKER` environment variables
* `kubernetes` - presents a filesystem hierarchy of pods, containers, and persistent volume claims
  - uses contexts from `~/.kube/config`
* `aws` - presents a filesystem hierarchy for EC2 and S3
  - uses `AWS_SHARED_CREDENTIALS_FILE` environment variable or `$HOME/.aws/credentials` and `AWS_CONFIG_FILE` environment variable or `$HOME/.aws/config` to find profiles and configure the SDK
  - IAM roles are supported when configured as described [here](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-role.html). Note that currently `region` will also need to be specified with the profile.
  - if using MFA, `wash` will prompt for it on standard input. Credentials are valid for 1 hour. They are cached under `wash/aws-credentials` in your [user cache directory](#user-cache-directory) so they can be re-used across server restarts. `wash` may have to re-prompt for a new MFA token in response to navigating the `wash` environment to authorize a new session.

[External plugins](https://github.com/puppetlabs/wash/tree/master/docs/external_plugins)

* `wash` allows for easy creation of out-of-process plugins using any language you want, from `bash` to `go` or anything in-between!
* `wash` handles the plugin lifecycle. it invokes your plugin with a certain calling convention; all you have to do is supply the business logic
* users interact with external plugins the exact same way as core plugins; they are first-class citizens

For more information about future direction, see our [Roadmap](https://github.com/puppetlabs/wash#roadmap)!