# wash (Wide Area SHell)

[![GitHub release](https://img.shields.io/github/release/puppetlabs/wash.svg)](https://github.com/puppetlabs/wash/releases/)[![Build Status](https://travis-ci.com/puppetlabs/wash.svg)](https://travis-ci.com/puppetlabs/wash)[![GoDoc](https://godoc.org/github.com/puppetlabs/wash?status.svg)](https://godoc.org/github.com/puppetlabs/wash)

`wash` helps you deal with all your remote or cloud-native infrastructure using the UNIX-y patterns and tools you already know and love!

_TBD: Screencast_

Exploring, understanding, and inspecting modern infrastructure should be simple and straightforward. Whether it's containers, VMs, network devices, IoT stuff, or anything in between...they all have different ways of enumerating what you have, getting a stream of output, running commands, etc. Every vendor has its own tools and APIs that expose these features, each one different, each one bespoke. Thus, they are difficult to compose together to solve higher-level problems. And that's no fun at all!

[UNIX's philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Origin) and abstractions have worked for decades. They're pretty good, and more importantly, they're _familiar_ to millions of people. `wash` intends to apply those same philosophies and abstractions to modern, distributed infrastructure: With `wash`, we want to:

* make navigating stuff like servers, containers, or APIs as easy as navigating a local filesystem
* make scripting across your new-fangled infrastructure as easy as writing a local shell script
* render into text that which can be rendered into text (cuz text is a universal interface!) for easy viewing, editing, and UNIXy slicing-and-dicing
* build new versions of basic, UNIX tools to support the above goals (but reuse existing ones if they work!)

## Features

We've implemented some neat features inside of `wash` to support the above goals:

* The `wash` daemon
    * presents a FUSE filesystem hierarchy for all of your resources, letting you navigate them in normal, filesystem-y ways
    * preserves history of all executed commands, facilitating debugging
    * serves up an HTTP API for everything
    * caches information, for better performance

* Primitives - the basic building blocks that form the guts of `wash`, and dictate what kinds of things you can do to all the resources `wash` knows about
    * `list` - lets you ask any resource what's contained inside of it, and what primitives it supports. 
        - _e.g. listing a Kubernetes pod returns its constituent containers_
    * `metadata` - returns the metadata for any resource
        - _e.g. you can use this to get all the metadata for a docker container, or a file in an S3 bucket, etc._
    * `read` - lets you read the contents of a given resource
        - _e.g. represent an EC2 instance's console output as a regular file you can open in a regular editor_
    * `stream` - gives you streaming-read access to a resource
        - _e.g. to let you follow a container's output as its running_
    * `exec` - lets you execute a command against a resource
        - _e.g. run a shell command inside a container, or on an EC2 vm, or on a routerOS device, etc._

* CLI tools
    * `wash ls` - a version of `ls` that uses our API to enhance directory listings with `wash`-specific info
        - _e.g. show you what primitives are supported for each resource_
    * `wash meta` - uses the `metadata` primitive to emit a resource's metadata to standard out
    * `wash exec` - uses the `exec` primtitive to let you invoke commands against resources

* Core plugins (see the _Roadmap_ below for more details)
    * `docker` - presents a filesystem hierarchy of containers and volumes
        - found from the local socket or via `DOCKER` environment variables
    * `kubernetes` - presents a filesystem hierarchy of pods, containers, and persistent volume claims
        - uses contexts from `~/.kube/config`
    * `aws` - presents a filesystem hierarchy for EC2 and S3
        - uses `AWS_SHARED_CREDENTIALS_FILE` environment variable or `$HOME/.aws/credentials`

* [External plugins](https://github.com/puppetlabs/wash/tree/master/docs/external_plugins)
    * `wash` allows for easy creation of out-of-process plugins using any language you want, from `bash` to `go` or anything in-between!
    * `wash` handles the plugin lifecycle. it invokes your plugin with a certain calling convention; all you have to do is supply the business logic
    * users interact with external plugins the exact same way as core plugins; they are first-class citizens

## Installation

### Binaries

_TBD: See [GitHub releases](https://github.com/puppetlabs/wash/releases)._

### From Source

Clone repo and within it run `go install`.

Ensure `$GOPATH/bin` is part of `$PATH`.

> Requires golang 1.11+.

### Additional macOS Setup

> If using iTerm2 and ZSH, we recommend installing [ZSH's iTerm2 shell integration](https://www.iterm2.com/documentation-shell-integration.html) to avoid [issue#84](https://github.com/puppetlabs/wash/issues/84).

Obtain FUSE for OSX [here](https://osxfuse.github.io/).

Add your mount directory to Spotlight's list of excluded directories to avoid heavy load.

## Usage

Mount `wash`'s filesystem and API server with
```
wash server mnt
```
In another shell, navigate to `mnt` to view available resources.

See available subcommands - such as `ls` and `exec` - with
```
wash help
```

_TBD: run docker, examine system_

All operations will have their activity recorded to journals in `wash/activity` under your user cache directory, identified by process ID and executable name. The user cache directory will be `$XDG_CACHE_HOME` or `$HOME/.cache` on Unix systems, `$HOME/Library/Caches` on macOS, and `%LocalAppData%` on Windows.

When done, `cd` out of `mnt`, then run `umount mnt` or `Ctrl-C` the server process.

## Known Issues

### On macOS

If the `wash` daemon exits with a exit status of 255, that typically means that `wash` couldn't load the FUSE extensions. MacOS only allows for a certain (small) number of virtual devices on the system, and if all available slots are taken up by other programs then we won't be able to run.

The biggest culprit is VirtualBox, which creates many virtual devices and keeps them resident even if the program isn't running. Currently, we recommend uninstalling VirtualBox to make room for `wash`.

More information in [this github issue for osxfuse](https://github.com/osxfuse/osxfuse/issues/358).

## Roadmap

[ ] Swagger docs for API server

## Contributing

We'd love to get contributions from you! For a quick guide, take a look at our guide to [contributing](./CONTRIBUTING.md).
