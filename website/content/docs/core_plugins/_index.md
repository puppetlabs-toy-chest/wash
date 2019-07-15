+++
title= "Core Plugins"
+++

## Working on core plugins

The [plugin] package defines a set of interfaces that a plugin can implement to enable specific behaviors.
- [Parent](https://godoc.org/github.com/puppetlabs/wash/plugin#Parent) can list its children and is presented as a directory in the Wash filesystem.
- [Readable](https://godoc.org/github.com/puppetlabs/wash/plugin#Readable) can retrieve the entire contents of a file and makes it available to read via standard system calls to the filesystem.
- [Streamable](https://godoc.org/github.com/puppetlabs/wash/plugin#Streamable) can provide a stream of updates on an entry. That may be events, log entries, or new writes to a file.
- [Execable](https://godoc.org/github.com/puppetlabs/wash/plugin#Execable) can execute an arbitrary command on a remote system. This is currently assumed to be a POSIX system, but in the future will be extended to differentiate between systems so we can use different commands as needed.

Each entry must implement [plugin.Entry](https://godoc.org/github.com/puppetlabs/wash/plugin#Entry), which is a sealed interface that can only be satisfied by [plugin.NewEntry](https://godoc.org/github.com/puppetlabs/wash/plugin#NewEntry).

Each entry that implements the `Parent` interface must provide a schema for its children.

Use [activity.Record](https://godoc.org/github.com/puppetlabs/wash/activity) for all plugin-related logging. Each plugin method that Wash calls is passed a `context.Context` object that is initialized with a Journal ID for use with `activity.Record`.

TIP: The [transport] package contains useful helpers for common methods of executing commands on a remote system. Currently it only supports SSH.

TIP: The [volume] package contains useful helpers that can enumerate a given volume's directories and files.

TIP: If there will only ever be one instance of the entry type - such as a named directory that's a container for a specific type of thing like EC2 instances - then use the [IsSingleton()](https://godoc.org/github.com/puppetlabs/wash/plugin#EntrySchema.IsSingleton) method when constructing the schema.


## How to create a new core plugin

Create a new directory in [plugin] for the plugin.

Create an object that implements the [Root](https://godoc.org/github.com/puppetlabs/wash/plugin#Root) interface. This would typically be in a `root.go` file. See [docker/root.go](https://github.com/puppetlabs/wash/blob/master/plugin/docker/root.go) for an example.

NOTE: The `Init` method initializes the Root object's `EntryBase` configuration and any credentials.

### Extending the plugin

Each entry in the plugin's hierarchy should be a new type. This pattern's adopted by the existing core plugins (e.g. [ec2Instance](https://github.com/puppetlabs/wash/blob/master/plugin/aws/ec2Instance.go) in AWS; [container](https://github.com/puppetlabs/wash/blob/master/plugin/docker/container.go) in Docker). It is meant to make your plugin modular and easier to maintain.

- Entries with children ("directories") should implement the `Parent` interface.
- Entries with content should implement `Readable`.
- Log-type entries should implement `Streamable` to expose a stream of new data.
- Entries that execute commands should implement the `Execable` interface.

TIP: The [volume.FS](https://godoc.org/github.com/puppetlabs/wash/volume#NewFS) helper can be used to expose an `Execable` entry's filesystem. It mounts the entire filesystem, and supports a configurable search depth; set it low if `Exec` operations for your plugin are fast, high if they're slow so that more of the filesystem is discovered in each batch.

[plugin]: https://godoc.org/github.com/puppetlabs/wash/plugin
[transport]: https://godoc.org/github.com/puppetlabs/wash/transport
[volume]: https://godoc.org/github.com/puppetlabs/wash/volume