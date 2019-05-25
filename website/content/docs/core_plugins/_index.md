+++
title= "Core Plugins"
+++

## How to create a new core plugin

The [plugin] package defines a set of interfaces that a plugin can implement to enable specific behaviors.

### Starting a plugin

Create a new directory in [plugin] for the plugin.

Create an object that implements the `Root` interface. This would typically be in a `root.go` file. See [docker/root.go](https://github.com/puppetlabs/wash/blob/master/plugin/docker/root.go) for an example.

NOTE: The `Init` method initializes the Root object's `EntryBase` configuration and any credentials.

### Extending the plugin

Each entry in the plugin's hierarchy should be a new type. This pattern's adopted by the existing core plugins (e.g. [ec2Instance](https://github.com/puppetlabs/wash/blob/master/plugin/aws/ec2Instance.go) in AWS; [container](https://github.com/puppetlabs/wash/blob/master/plugin/docker/container.go) in Docker). It is meant to make your plugin modular and easier to maintain.

- Entries with children ("directories") should implement the `Parent` interface.
- Entries with content should implement `Readable`.
- Log-type entries should implement `Streamable` to expose a stream of new data.
- Entries that execute commands should implement the `Execable` interface.

TIP: Use [activity.Record](https://github.com/puppetlabs/wash/blob/master/activity/core.go) for all plugin-related logging.

TIP: The [volume] package contains useful helpers that can enumerate a given volume's directories and files.

TIP: The [volume.FS](https://github.com/puppetlabs/wash/blob/master/volume/fs.go) helper can be used to expose an `Execable` entry's filesystem. Currently only `/var/log` is mounted.

[plugin]: https://github.com/puppetlabs/wash/tree/master/plugin
[volume]: https://github.com/puppetlabs/wash/tree/master/volume