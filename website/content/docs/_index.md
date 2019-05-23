+++
title= "Wash Documentation"
+++

* [Wash Commands](#wash-commands)
  * [wash](#wash)
  * [wash clear](#wash-clear)
  * [wash exec](#wash-exec)
  * [wash find](#wash-find)
  * [wash history](#wash-history)
  * [wash info](#wash-info)
  * [wash list](#wash-list-ls)
  * [wash meta](#wash-meta)
  * [wash ps](#wash-ps)
  * [wash server](#wash-server)
  * [wash tail](#wash-tail)
* [Config] (#config)
* [Core Plugins](#core-plugins)
  * [AWS](#aws)
  * [Docker](#docker)
  * [Kubernetes](#kubernetes)
* [Plugin Concepts](#plugin-concepts)
  * [➠External plugins]
  * [➠Go Plugins]
  * [➠Server API]

## Wash Commands

Wash commands aim to be well-documented in the tool. Try `wash help` and `wash help <subcommand>` for specific options.

Most commands operate on Wash resources, which are addressed by their path in the filesystem.

### wash

The `wash` command can be invoked on its own to enter a Wash shell.

Invoking `wash` starts the daemon as part of the process, then enters your current system shell with shortcuts configured for wash subcommands. All the [`wash server`](#wash-server) settings are also supported with `wash`.

### wash clear

Wash caches most operations. If the resource you're querying appears out-of-date, use this subcommand to reset the cache for resources at or contained within the specified path. Defaults to the current directory if a path is not specified.

### wash exec

For a Wash resource that implements the ability to execute a command, run the specified command and arguments. The results will be forwarded from the target on stdout, stderr, and exit code.

### wash find

Recursively descends the directory tree of the specified path, evaluating an `expression` composed of `primaries` and `operands` for each entry in the tree.

### wash history

Wash maintains a history of commands executed through it. Print that command history, or specify an `id` to print a log of activity related to a particular command.

### wash info

Print all info Wash has about the specified path, including filesystem attributes and metadata.

### wash list/ls

Lists the resources at the indicated path.

### wash meta

Prints the entry's metadata. By default, meta prints the full metadata as returned by the metadata endpoint. Specify the `--attribute` flag to print the meta attribute instead.

### wash ps

Captures /proc/*/{cmdline,stat,statm} on each node by executing 'cat' on them. Collects the output
to display running processes on all listed nodes. Errors on paths that don't implement exec.

### wash server

Initializes all of the plugins, then sets up the Wash daemon (its API and FUSE servers). To stop it, make sure you're not using the filesystem at the specified mountpoint, then enter Ctrl-C.

Server API docs can be found [here](/wash/docs/api). The server config is described in the [`config`](#config) section.

### wash tail

Output any new updates to files and/or resources (that support the stream action). Currently requires the '-f' option to run. Attempts to mimic the functionality of `tail -f` for remote logs.

## Config

The Wash config file is located at `~/.puppetlabs/wash/wash.yaml`, and can be used to configure the [`wash-server`](#wash-server). You can override this location via the `config-file` flag.

Below are all the configurable options.

* `logfile` - The location of the server's log file (default `stdout`)
* `loglevel` - The server's loglevel (default `info`)
* `cpuprofile` - The location that the server's CPU profile will be written to (optional)
* `external-plugins` - The external plugins that will be loaded. See [➠External Plugins]
* `socket` - The location of the server's socket file (default `<user_cache_dir>/wash/wash-api.sock`)

All options except for `external-plugins` can be overridden by setting the `WASH_<option>` environment variable.

NOTE: Do not override `socket` in a config file. Instead, override it via the `WASH_socket` environment variable. Otherwise, Wash's subcommands will not be able to interact with the server because they cannot access the socket.

## Core Plugins

### AWS

- EC2 and S3
- uses `AWS_SHARED_CREDENTIALS_FILE` environment variable or `$HOME/.aws/credentials` and `AWS_CONFIG_FILE` environment variable or `$HOME/.aws/config` to find profiles and configure the SDK.
- IAM roles are supported when configured as described here. Note that currently region will also need to be specified with the profile.
- if using MFA, wash will prompt for it on standard input. Credentials are valid for 1 hour. They are cached under `wash/aws-credentials` in your user cache directory so they can be re-used across server restarts. wash may have to re-prompt for a new MFA token in response to navigating the wash environment to authorize a new session.
- supports streaming, and remote command execution via ssh
- supports full metadata for S3 content

### Docker

- containers and volumes
- found from the local socket or via `DOCKER` environment variables
- supports streaming, and remote command execution

### Kubernetes

- pods, containers, and persistent volume claims
- uses contexts from `~/.kube/config`
- supports streaming, and remote command execution
- supports listing of volume contents

## Plugin Concepts

Wash's plugin system is designed around a set of primitives that resources can implement. A plugin requires a root that can list things it contains, and implements a tree structure under that where each node represents a resource or an arbitrary grouping. Wash will translate that tree structure into a file hierarchy.

Wash supports the following primitives:

* `list` - lets you ask any resource what's contained inside of it, and what primitives it supports.
  - _e.g. listing a Kubernetes pod returns its constituent containers_
* `read` - lets you read the contents of a given resource
  - _e.g. represent an EC2 instance's console output as a regular file you can open in a regular editor_
* `stream` - gives you streaming-read access to a resource
  - _e.g. to let you follow a container's output as its running_
* `exec` - lets you execute a command against a resource
  - _e.g. run a shell command inside a container, or on an EC2 vm, or on a routerOS device, etc._

Primitives can be accessed programmatically via the Wash API, or on the CLI via `wash` subcommands and filesystem interactions.

For more on implementing plugins, see:

* [➠External plugins]
* [➠Go Plugins]
* [➠Server API]

[➠External plugins]: /wash/docs/external_plugins
[➠Go plugins]: /wash/docs/go_plugins
[➠Server API]: /wash/docs/api