# wash (Wide Area SHell)

`wash` helps you deal with all your remote or cloud-native infrastructure using the UNIX-y patterns and tools you already know and love!

Exploring, understanding, and inspecting modern infrastructure should be simple and straightforward. Whether it's containers, VMs, network devices, IoT stuff, or anything in between...they all have different ways of enumerating what you have, getting a stream of output, running commands, etc. Every vendor has its own tools and APIs that expose these features, each one different, each one bespoke. Thus, they are difficult to compose together to solve higher-level problems. And that's no fun at all!

[UNIX's philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Origin) and abstractions have worked for decades. They're pretty good, and more importantly, they're _familiar_ to millions of people. `wash` intends to apply those same philosophies and abstractions to modern, distributed infrastructure: With `wash`, we want to:

* make navigating stuff like servers, containers, or APIs as easy as navigating a local filesystem
* make scripting across your new-fangled infrastructure as easy as writing a local shell script
* render into text that which can be rendered into text (cuz text is a universal interface!) for easy viewing, editing, and UNIXy slicing-and-dicing
* build new versions of basic, UNIX tools to support the above goals (but reuse existing ones if they work!)

## Usage

This prototype is built as a FUSE filesystem and API server. It currently supports
- viewing running containers and volumes in Docker (found from the local socket or via DOCKER environment variables)
- viewing running containers and persistent volume claims in Kubernetes (uses contexts from `~/.kube/config`)

> Requires golang 1.11+.

Mount the filesystem and API server with
```
go run wash.go server mnt
```

In another shell, navigate it at `mnt`. When done, `cd` out of `mnt`, then run `umount mnt` or `Ctrl-C` the server process.

Operations that work:
- `ls`
- `cat`
- `vim`
- `tail [-f]`
- `stat` (kind of, information's not very useful)

Wash also comes with some custom commands you can run. These are:
- `wash ls <path>` (a custom `ls` command that displays the name, creation time, and available actions on a remote resource)
- `wash meta <path>` (outputs a remote resource's metadata as JSON)
- `wash exec <path> <cmd> [<arg>...]` (executes cmd with args on a remote compute resource)

Currently, these commands can only be run in the project directory. Thus, if you'd like to view the metadata of a Docker container, you'd need to do
```
go run wash.go meta mnt/docker/containers/<container_name>
```

The API server exposes advanced capabilities on resources; Swagger docs TODO.

All operations will have their activity recorded to journals in `wash/activity` under your user cache directory, identified by process ID and executable name. The user cache directory will be `$XDG_CACHE_HOME` or `$HOME/.cache` on Unix systems, `$HOME/Library/Caches` on macOS, and `%LocalAppData%` on Windows.

### macOS Setup

> If using iTerm2 and ZSH, we recommend installing [ZSH's iTerm2 shell integration](https://www.iterm2.com/documentation-shell-integration.html) to avoid [issue#84](https://github.com/puppetlabs/wash/issues/84).

Obtain FUSE for OSX [here](https://osxfuse.github.io/).

Add your mount directory to Spotlight's list of excluded directories to avoid heavy load.

#### Known issues on Mac

If the `wash` daemon exits with a exit status of 255, that typically means that `wash` couldn't load the FUSE extensions. MacOS only allows for a certain (small) number of virtual devices on the system, and if all available slots are taken up by other programs then we won't be able to run.

The biggest culprit is VirtualBox, which creates many virtual devices and keeps them resident even if the program isn't running. Currently, we recommend uninstalling VirtualBox to make room for `wash`.

More information in [this github issue for osxfuse](https://github.com/osxfuse/osxfuse/issues/358).

## Design

Several principles that inform the design of *wash* are:
- Multiple ways to get data, but consistent language within the tool. i.e. may search for a database by saying type is 'db' or 'database', but the tool will always refer to them by 'database'.
- Rich shell experience.
- Store everything for future use.

Examples of things we think *wash* could do: [EXAMPLES.md](./EXAMPLES.md)

Taxonomy of remote resources: [TAXONOMY.md](./TAXONOMY.md)

## Contributing

We'd love to get contributions from you! For a quick guide, take a look at our guide to [contributing](./CONTRIBUTING.md).
