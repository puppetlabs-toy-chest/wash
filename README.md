# wash (Wide Area SHell)

A cloud-native shell for bringing remote infrastructure to your terminal.

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

## Design

Several principles that inform the design of *wash* are:
- Multiple ways to get data, but consistent language within the tool. i.e. may search for a database by saying type is 'db' or 'database', but the tool will always refer to them by 'database'.
- Rich shell experience.
- Store everything for future use.

Examples of things we think *wash* could do: [EXAMPLES.md](./EXAMPLES.md)

Taxonomy of remote resources: [TAXONOMY.md](./TAXONOMY.md)

## Contributing

We'd love to get contributions from you! For a quick guide, take a look at our guide to [contributing](./CONTRIBUTING.md).
