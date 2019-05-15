+++
title= "Go Plugins"
+++

## How to create a new Go plugin

The [plugin] package defines a set of interfaces that a plugin can implement to enable specific behaviors.

### Starting a plugin

Create a new directory in [plugin] for the plugin.

Create an object that implements the `Root` profile, as in it has:

- a `Name` method to determine the name of the plugin when mounted
- an `Init` method that loads and validates credentials
- a `List` method that lists the first-level resources of the plugin

### Extending the plugin

Add new types representing the types of resources the plugin can list.

- To make them appear as a directory, implement the `Group` interface.
- Resources with metadata should implement the `Resource` interface.
- To give files content, implement the `Readable` interface.
- Log-type files can implement the `Pipe` interface to expose a stream of new data.
- Files and directories that exist on a remote system or volume should implement the `File` interface. Additionally the [volume] package provides helpers for representing these types.
- To support executing commands on a resource, implement the `Execable` interface.

Tip: base new resources on the `EntryBase` struct to simplify creating them and enable caching.

[plugin]: https://github.com/puppetlabs/wash/tree/master/plugin
[volume]: https://github.com/puppetlabs/wash/tree/master/volume