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
  * [Attributes/Metadata](#attributes-metadata)
  * [➠External plugins]
  * [➠Core Plugins]
  * [➠Server API]

## Wash Commands

Wash commands aim to be well-documented in the tool. Try `wash help` and `wash help <subcommand>` for specific options.

Most commands operate on Wash resources, which are addressed by their path in the filesystem.

### wash

The `wash` command can be invoked on its own to enter a Wash shell.

Invoking `wash` starts the daemon as part of the process, then enters your current system shell with shortcuts configured for wash subcommands. All the [`wash server`](#wash-server) settings are also supported with `wash` except `socket`; `wash` ignores that setting and creates a temporary location for the socket.

### wash clear

Wash caches most operations. If the resource you're querying appears out-of-date, use this subcommand to reset the cache for resources at or contained within the specified path. Defaults to the current directory if a path is not specified.

### wash exec

For a Wash resource that implements the ability to execute a command, run the specified command and arguments. The results will be forwarded from the target on stdout, stderr, and exit code.

### wash find

Recursively descends the directory tree of the specified paths, evaluating an `expression` composed of `primaries` and `operands` for each entry in the tree.

### wash history

Wash maintains a history of commands executed through it. Print that command history, or specify an `id` to print a log of activity related to a particular command.

### wash info

Print all info Wash has about the specified path, including filesystem attributes and metadata.

### wash list/ls

Lists the resources at the indicated path.

### wash meta

Prints the entry's metadata. By default, meta prints the full metadata as returned by the metadata endpoint. Specify the `--attribute` flag to instead print the meta attribute, a (possibly) reduced set of metadata that's returned when entries are enumerated.

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

All options except for `external-plugins` can be overridden by setting the `WASH_<option>` environment variable with option converted to ALL CAPS.

NOTE: Do not override `socket` in a config file. Instead, override it via the `WASH_SOCKET` environment variable. Otherwise, Wash's subcommands will not be able to interact with the server because they cannot access the socket.

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

Everything is an entry in Wash. This includes resources like containers and volumes; organizational groups like the containers directory in the Docker plugin; read-only files like the metadata.json files for EC2 instances; and even non-infrastructure related things like Goodreads books, cooking recipes, breweries, Fandango theaters and movies, etc. (Yes, you can write a Wash plugin for Fandango. In fact, you can write a Wash plugin for anything that you can model as a filesystem.)

Plugins have their own file hierarchy that's described by a tree structure, where the entries are the nodes. Additionally, internal nodes (i.e. entries with children) are classified as "parents." Parents represent the "directories" of the plugin's filesystem, while everything else is a "file."

Plugins are written in top-down fashion, starting with the root. All entries (and their paths) are referenced by their canonical name (`cname`). The `cname` consists of the entry's reported name with all `/`'es replaced by `#`. You can override this with your own slash replacer.

Wash entries can support the following actions:

* `list` - lists an entry's children, including their supported actions
  - _e.g. listing a Kubernetes pod returns its constituent containers_
* `read` - lets you read the entry's content
  - _e.g. represent an EC2 instance's console output as a regular file you can open in a regular editor_
* `stream` - gives you streaming-read access to an entry
  - _e.g. to let you follow a container's output as its running_
* `exec` - lets you execute a command against an entry
  - _e.g. run a shell command inside a container, or on an EC2 vm, or on a routerOS device, etc._

Actions can be invoked programmatically via the Wash API, or on the CLI via `wash` subcommands and filesystem interactions.

For more on implementing plugins, see:

* [➠External plugins]
* [➠Core Plugins]
* [➠Server API]

[➠External plugins]: /wash/docs/external_plugins
[➠Core plugins]: /wash/docs/core_plugins
[➠Server API]: /wash/docs/api

NOTE: We recommend that you read the `Attributes/Metadata` section before reading the plugin tutorials to take full advantage of Wash's capabilities, especially that of `wash find`'s.

### Attributes/Metadata

All entries have metadata, which is a JSON object containing a complete description of the entry. For example, a Docker container's metadata includes its labels, its state, its start time, the image it was built from, its mounted volumes, etc. [`wash find`](#wash-find) can filter on this metadata. In our example, you can use `find docker/containers -daystart -fullmeta -m .state .startedAt -{1d} -a .status running` to see a list of all running containers that started today (try it out!). Thus, metadata filtering is powerful. However, it also requires the user to query an entry's metadata to construct the filter. Creating a filter on the same property that's shared by many different kinds of entries is repetitive, error-prone, and an obvious candidate for usability improvement. For example, metadata filtering gets annoying when you are trying to filter on an EC2 instance's/Docker container's/Kubernetes pod's state due to the structural differences in their metadata (e.g. an EC2 instance's state is contained in the `.state.name` key, while a Kubernetes pod's state is contained in the `.status.phase` key). Metadata filtering is also slow. It requires O(N) API requests, where N is the number of visited entries.

To make `wash find`'s filtering less tedious and better performing, entries can also have attributes. The attributes represent common metadata properties that people filter on. Currently, these are the traditional `ctime`, `mtime`, `atime`, `size`, and `mode` filesystem attributes, along with a special `meta` attribute representing a subset of the entry's metadata (useful for fast metadata filtering). The attributes are fetched in bulk when the entry's parent is listed. Typically, the bulk fetch is done through an API's `list` endpoint. This endpoint returns an array of JSON objects representing the entries. The `meta` attribute is set to this JSON object while the remaining attributes are parsed from the object's fields. For example, `list docker/containers` will fetch all of your containers by querying Docker's `/containers/json` endpoint. That endpoint's response is then used to create the container entry objects, where each container entry's `meta` attribute is set to a `/containers/json` object and the containers' `ctime`/`mtime` attributes are parsed from it.

NOTE: _All_ attributes are optional, so set the ones that you think make sense. For example, if the `mode` or `size` attributes don't make sense for your entry, then feel free to ignore them. However, we recommend that you try to set the `meta` attribute when you can to take advantage of metadata filtering.

NOTE: We plan on adding more attributes depending on user feedback (e.g. like `state` and `labels`). Thus if you find yourself metadata-filtering on a common property across a bunch of different entries, then please feel free to file an issue so we can consider adding that property as an attribute (and as a corresponding `wash find` primary).