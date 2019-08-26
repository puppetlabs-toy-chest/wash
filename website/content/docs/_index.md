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
  * [wash list/ls](#wash-list-ls)
  * [wash meta](#wash-meta)
  * [wash ps](#wash-ps)
  * [wash server](#wash-server)
  * [wash stree](#wash-stree)
  * [wash tail](#wash-tail)
  * [wash validate](#wash-validate)
* [Config](#config)
  * [wash.yaml](#wash-yaml)
  * [wash shell](#wash-shell)
* [Core Plugins](#core-plugins)
  * [AWS](#aws)
  * [Docker](#docker)
  * [GCP](#gcp)
  * [Kubernetes](#kubernetes)
* [Plugin Concepts](#plugin-concepts)
  * [Plugin Debugging](#plugin-debugging)
  * [Attributes/Metadata](#attributes-metadata)
  * [Entry Schemas](#entry-schemas)
  * [➠External plugins]
  * [➠Core Plugins]
  * [➠Server API]
* [Analytics](#analytics)
  * [What data does Wash collect?](#what-data-does-wash-collect)
  * [Why does Wash collect data?](#why-does-wash-collect-data)
  * [How can I opt out of Wash data collection?](#how-can-i-opt-out-of-wash-data-collection)

## Wash Commands

Wash commands aim to be well-documented in the tool. Try `wash help` and `wash help <command>` for specific options.

Most commands operate on Wash resources, which are addressed by their path in the filesystem.

### wash

The `wash` command can be invoked on its own to enter a Wash shell.

Invoking `wash` starts the daemon as part of the process, then enters your current system shell with shortcuts configured for Wash commands. All the [`wash server`](#wash-server) settings are also supported with `wash` except `socket`; `wash` ignores that setting and creates a temporary location for the socket.

### wash clear

Wash caches most operations. If the resource you're querying appears out-of-date, use this command to reset the cache for resources at or contained within the specified path. Defaults to the current directory if a path is not specified.

### wash exec

For a Wash resource that implements the ability to execute a command, run the specified command and arguments. The results will be forwarded from the target on stdout, stderr, and exit code.

### wash find

Recursively descends the directory tree of the specified paths, evaluating an `expression` composed of `primaries` and `operands` for each entry in the tree.

### wash history

Wash maintains a history of commands executed through it. Print that command history, or specify an `id` to print a log of activity related to a particular command.

Journals are stored in `wash/activity` under your user cache directory, identified by process ID and executable name. The user cache directory is `$XDG_CACHE_HOME` or `$HOME/.cache` on Unix systems, `$HOME/Library/Caches` on macOS, and `%LocalAppData%` on Windows.

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

Initializes all of the plugins, then sets up the Wash daemon (its API and [FUSE](https://en.wikipedia.org/wiki/Filesystem_in_Userspace) servers). To stop it, make sure you're not using the filesystem at the specified mountpoint, then enter Ctrl-C.

Server API docs can be found [here](api). The server config is described in the [`config`](#config) section.

### wash stree

Displays the entry's stree (schema-tree), which is a high-level overview of the entry's hierarchy. Non-singleton types are bracketed with "[]".

### wash tail

Output any new updates to files and/or resources (that support the stream action). Currently requires the '-f' option to run. Attempts to mimic the functionality of `tail -f` for remote logs.

### wash validate

Validates an external plugin, using it's schema to limit exploration. The plugin can be one you've configured in Wash's config file, or it can be a script to load as an external plugin. Plugin-specific config from Wash's config file will be used. The Wash daemon does not need to be running to use this command.

Validate starts from the plugin root and does a breadth-first traversal of the plugin hierarchy, invoking all supported methods on an example at each level. If the plugin provides a schema, it will be used to explore one example of each type of entry. Exploration can be stopped with Ctrl-C when needed.

Each line represents validation of an entry type. The `lrsx` fields represent support for `list`, `read`, `stream`, and `execute` methods respectively, with '-' representing lack of support for a method.

## Config

### wash.yaml

The Wash config file is located at `~/.puppetlabs/wash/wash.yaml`, and can be used to configure the [`wash-server`](#wash-server). You can override this location via the `config-file` flag.

Below are all the configurable options.

* `logfile` - The location of the server's log file (default `stdout`)
* `loglevel` - The server's loglevel (default `info`)
* `cpuprofile` - The location that the server's CPU profile will be written to (optional)
* `external-plugins` - The external plugins that will be loaded. See [➠External Plugins]
* `plugins` - A list of core plugins to enable. If omitted or empty, it will load all available plugins.
* `socket` - The location of the server's socket file (default `<user_cache_dir>/wash/wash-api.sock`)

All options except for `external-plugins` can be overridden by setting the `WASH_<option>` environment variable with option converted to ALL CAPS.

NOTE: Do not override `socket` in a config file. Instead, override it via the `WASH_SOCKET` environment variable. Otherwise, Wash's commands will not be able to interact with the server because they cannot access the socket.

### wash shell

Wash uses your system shell to provide the shell environment. It determines this using the `SHELL` environment variable or falls back to `/bin/sh`, so if you'd like to specify a particular shell set the `SHELL` environment variable before starting Wash.

For some shells, Wash provides a customized environment. Please [file an issue](https://github.com/puppetlabs/wash/issues/new?assignees=&labels=Feature&template=feature-request.md) if you'd like to add support for new shells.

Wash currently provides a customized environment for
- `bash`
- `zsh`

Customized environments alias Wash subcommands to save typing out `wash <subcommand>` so they feel like shell builtins. If you want to use an executable or builtin Wash has overridden, please use its full path or the `builtin` command.

Customized environments also supports reading `~/.washenv` and `~/.washrc` files. These files are loaded as follows:
1. If running Wash non-interactively (by piping `stdin` or passing the `-c` option)
   1. If `~/.washenv` does not exist, load the shell's default non-interactive config (such as `.zshenv` or from `BASH_ENV`)
   2. Configure subcommand aliases
   3. If `~/.washenv` exists, load it
2. If running Wash interactively
   1. Do all non-interactive config above
   2. If `~/.washrc` does not exist, load the shell's default interactive config (such as `.bash_profile` or `.zshrc`)
   3. Configure the command prompt
   4. If `~/.washrc` exists, load it

For other shells, Wash creates executables for subcommands and does no other customization.

## Core Plugins

### AWS

- EC2 and S3
- IAM roles are supported when configured as described here. Note that currently region will also need to be specified with the profile.
- if using MFA, Wash will prompt for it on standard input. Credentials are valid for 1 hour. They are cached under `wash/aws-credentials` in your user cache directory so they can be re-used across server restarts. Wash may have to re-prompt for a new MFA token in response to navigating the Wash environment to authorize a new session.
- supports streaming, and remote command execution via ssh
- supports full metadata for S3 content

The AWS plugin reads the `AWS_SHARED_CREDENTIALS_FILE` environment variable or `$HOME/.aws/credentials` and `AWS_CONFIG_FILE` environment variable or `$HOME/.aws/config` to find profiles and configure the SDK. The profiles it lists can be limited by adding
```
aws:
  profiles: [profile_1, profile_2]
```
to Wash's [config file](#config).

#### Exec

The `exec` method for AWS uses SSH. It will look up port, user, and other configuration by exact hostname match from default SSH config files. If present, a local SSH agent will be used for authentication.

Lots of SSH configuration is currently omitted, such as global known hosts files, finding known hosts from the config, identity file from config... pretty much everything but port and user from config as enumerated in https://github.com/kevinburke/ssh_config/blob/0.5/validators.go.

The known hosts file will be ignored if StrictHostKeyChecking=no, such as in
```
Host *.compute.amazonaws.com
  StrictHostKeyChecking no
```

### Docker

- containers and volumes
- found from the local socket or via `DOCKER` environment variables
- supports streaming, and remote command execution

### GCP

The GCP plugin follows https://cloud.google.com/docs/authentication/production to find your credentials:
- it will try `GOOGLE_APPLICATION_CREDENTIALS` as a service account file
- use your credentials in `$HOME/.config/gcloud/application_default_credentials.json`

The simplest way to set this up is with
```
gcloud init
gcloud auth application-default login
```

The GCP plugin will list all projects you have access to. The projects it lists can be limited by adding
```
gcp:
  projects: [project-1, project-2]
```
to Wash's [config file](#config). Project can be referenced either by name or project ID.

#### Exec

The Exec method mirrors running [`gcloud compute ssh`](https://cloud.google.com/sdk/gcloud/reference/compute/ssh). If not already present, it will generate a Google Compute-specific SSH key pair and known hosts file in your `~/.ssh` directory and ensure they're present on the machine you're trying to connect to. Your current `$USER` name will be used as the login user.

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

For entries that can be `read`, provide the size if you know it; otherwise Wash will provide a functional default and update the size when the entry has been `read`. Note that `find -size` will not include files with unknown size.

Actions can be invoked programmatically via the Wash API, or on the CLI via `wash` commands and filesystem interactions.

For more on implementing plugins, see:

* [➠External plugins]
* [➠Core Plugins]
* [➠Server API]

[➠External plugins]: external_plugins
[➠Core plugins]: core_plugins
[➠Server API]: api

NOTE: We recommend that you read the [Attributes/Metadata](#attributes/metadata) section before reading the plugin tutorials to take full advantage of Wash's capabilities, especially that of `wash find`'s.

### Plugin Debugging

Plugin-related activity is currently logged at `debug` level. You can control logging with the `loglevel` and `logfile` options to `wash` or `wash server`. So when developing a plugin, it's useful to start your shell with

```
wash --loglevel debug --logfile <file>
```

then `tail -f <file>` in another terminal to see what Wash is doing.

For external plugins, those logs will include the commands used for all invocations of your plugin script and the responses. For example

```
level=debug msg="Invoking /washreads/goodreads list /goodreads \\{\\\"userid\\\":\\\"12345678\\\"}"
level=debug msg="stdout: [{\"name\":\"read\",\"methods\":[\"list\"],\"attributes\":{\"meta\":{\"id\":\"53094184\",\"book_count\":\"494\",\"exclusive_flag\":\"true\",\"description\":\"\",\"sort\":\"\",\"order\":\"\",\"per_page\":\"\",\"display_fields\":\"\",\"featured\":\"true\",\"recommend_for\":\"false\",\"sticky\":\"\"}},\"state\":\"{\\\"type\\\":\\\"shelf\\\",\\\"name\\\":\\\"read\\\",\\\"userid\\\":\\\"16580428\\\",\\\"count\\\":494}\"},...]\n"
level=debug msg="stderr: something's happening"
```

Activity related to a specific operation are always available in the `history` entry for that operation. Given a history entry like
```
$ whistory
1  2019-06-20 13:47  wash whistory
2  2019-06-20 13:47  ls -pG goodreads
```
we can view activity related to that command with
```
$ whistory 2
Jun 20 13:47:09.212 FUSE: List /goodreads
Jun 20 13:47:09.212 Invoking /Users/michaelsmith/puppetlabs/washreads/goodreads list /goodreads \{\"userid\":\"16580428\"}
Jun 20 13:47:09.886 stdout: [{"name":"read","methods":["list"],"attributes":{"meta":{"id":"53094184","book_count":"494","exclusive_flag":"true","description":"","sort":"","order":"","per_page":"","display_fields":"","featured":"true","recommend_for":"false","sticky":""}},"state":"{\"type\":\"shelf\",\"name\":\"read\",\"userid\":\"16580428\",\"count\":494}"},...]
Jun 20 13:47:09.886 FUSE: Listed in /goodreads: [{Inode:0 Type:dir Name:read} {Inode:0 Type:dir Name:currently-reading} {Inode:0 Type:dir Name:to-read} {Inode:0 Type:dir Name:fantasy} {Inode:0 Type:dir Name:science-fiction}]
```

### Attributes/Metadata

All entries have metadata, which is a JSON object containing a complete description of the entry. For example, a Docker container's metadata includes its labels, its state, its start time, the image it was built from, its mounted volumes, etc. [`wash find`](#wash-find) can filter on this metadata. In our example, you can use `find docker/containers -daystart -fullmeta -m .state .startedAt -{1d} -a .status running` to see a list of all running containers that started today (try it out!). Thus, metadata filtering is powerful. However, it also requires the user to query an entry's metadata to construct the filter. Creating a filter on the same property that's shared by many different kinds of entries is repetitive, error-prone, and an obvious candidate for usability improvement. For example, metadata filtering gets annoying when you are trying to filter on an EC2 instance's/Docker container's/Kubernetes pod's state due to the structural differences in their metadata (e.g. an EC2 instance's state is contained in the `.state.name` key, while a Kubernetes pod's state is contained in the `.status.phase` key). Metadata filtering is also slow. It requires O(N) API requests, where N is the number of visited entries.

To make `wash find`'s filtering less tedious and better performing, entries can also have attributes. The attributes represent common metadata properties that people filter on. Currently, these are the traditional `crtime`, `mtime`, `ctime`, `atime`, `size`, and `mode` filesystem attributes, along with a special `meta` attribute representing a subset of the entry's metadata (useful for fast metadata filtering). The attributes are fetched in bulk when the entry's parent is listed. Typically, the bulk fetch is done through an API's `list` endpoint. This endpoint returns an array of JSON objects representing the entries. The `meta` attribute is set to this JSON object while the remaining attributes are parsed from the object's fields. For example, `list docker/containers` will fetch all of your containers by querying Docker's `/containers/json` endpoint. That endpoint's response is then used to create the container entry objects, where each container entry's `meta` attribute is set to a `/containers/json` object and the containers' `crtime`/`mtime` attributes are parsed from it.

NOTE: _All_ attributes are optional, so set the ones that you think make sense. For example, if the `mode` or `size` attributes don't make sense for your entry, then feel free to ignore them. However, we recommend that you try to set the `meta` attribute when you can to take advantage of metadata filtering.

NOTE: We plan on adding more attributes depending on user feedback (e.g. like `state` and `labels`). Thus if you find yourself metadata-filtering on a common property across a bunch of different entries, then please feel free to file an issue so we can consider adding that property as an attribute (and as a corresponding `wash find` primary).

### Entry Schemas

Entry schemas are a type-level overview of your plugin's hierarchy. They enumerate the kinds of things your plugins can contain, including what those things look like. For example, a Docker container's schema would answer questions like:

* Can I create multiple Docker containers?
* What's in a Docker container's metadata?
* What Wash actions does a Docker container support?
* If I `ls` a Docker container, what do I get?

These questions can be generalized to any Wash entry.

Entry schemas are a useful way to document your plugin's hierarchy without having to maintain a README. Users can view your hierarchy via the `stree` command. For example, if you invoke `stree docker` in a Wash shell (try it!), you should see something like

```
docker
├── containers
│   └── [container]
│       ├── log
│       ├── metadata.json
│       └── fs
│           ├── [dir]
│           │   ├── [dir]
│           │   └── [file]
│           └── [file]
└── volumes
    └── [volume]
        ├── [dir]
        │   ├── [dir]
        │   └── [file]
        └── [file]
```

(Your output may differ depending on the state of the Wash project, but it should be similarly structured).

Every node must have a label. The `[]` are printed for non-singleton nodes; they imply multiple instances of this thing. For example, `[container]` means that there will be multiple `container` instances under the `containers` directory. Similarly, `containers` means that there will be only one `containers` directory (i.e. that `containers` is a singleton). Singleton entries should typically use the entry's name as the label.

Entry schemas are also useful for optimizing `find`, especially when `find` is used for metadata filtering. Without entry schemas, for example, an EC2 instance query like `find aws -meta '.tags[?]' .key termination_date` would cause `find` to recurse into every entry in the `aws` plugin, including non-EC2 instance entries like S3 objects. With entry schemas, however, `find` would only recurse into those entries that will eventually lead to an EC2 instance. The latter is a significantly faster (and less expensive) operation, especially for large infrastructures.

## Analytics

Wash collects anonymous data about how you use it. You can opt out of providing this data.

### What data does Wash collect?
* Version of Wash
* User locale
* Architecture
* Method invocations (for core plugin entries only)
  * This includes any invocation of the `List`, `Exec`, `Read`, and `Stream` primitives
  * Also includes the entry's plugin

This data is associated with Bolt analytics' UUID (if available); otherwise, the data is associated with a random, non-identifiable user UUID.

### Why does Wash collect data?
Wash collects data to help us understand how it's being used and make decisions about how to improve it.

### How can I opt out of Wash data collection?
To disable the collection of analytics data add the following line to `~/.puppetlabs/wash/analytics.yaml`:

```
disabled: true
```

You can also disable the collection of analytics data by setting the `WASH_DISABLE_ANALYTICS` environment variable to `true` before starting up the Wash daemon.