# wash (Wide Area SHell)

_Badges: latest version, CI status, godoc reference

`wash` helps you deal with all your remote or cloud-native infrastructure using the UNIX-y patterns and tools you already know and love!

_TBD: Screencast_

Exploring, understanding, and inspecting modern infrastructure should be simple and straightforward. Whether it's containers, VMs, network devices, IoT stuff, or anything in between...they all have different ways of enumerating what you have, getting a stream of output, running commands, etc. Every vendor has its own tools and APIs that expose these features, each one different, each one bespoke. Thus, they are difficult to compose together to solve higher-level problems. And that's no fun at all!

[UNIX's philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Origin) and abstractions have worked for decades. They're pretty good, and more importantly, they're _familiar_ to millions of people. `wash` intends to apply those same philosophies and abstractions to modern, distributed infrastructure: With `wash`, we want to:

* make navigating stuff like servers, containers, or APIs as easy as navigating a local filesystem
* make scripting across your new-fangled infrastructure as easy as writing a local shell script
* render into text that which can be rendered into text (cuz text is a universal interface!) for easy viewing, editing, and UNIXy slicing-and-dicing
* build new versions of basic, UNIX tools to support the above goals (but reuse existing ones if they work!)

## Features (TBD)

`wash`'s interactions center around resources that implement common primitives. Those resources are implemented in plugins.

### Primitives

_Supersede TAXONOMY.md_

### Plugins

In the core distribution
- Docker: containers and volumes. Found from the local socket or via `DOCKER` environment variables.
- Kubernetes: containers and persistent volume claims. Uses contexts from `~/.kube/config`.
- AWS: EC2 instances and S3 buckets. Uses `AWS_SHARED_CREDENTIALS_FILE` environment variable or `$HOME/.aws/credentials`.

You can also create external plugins: [docs](https://github.com/puppetlabs/wash/tree/master/docs/external_plugins).

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
