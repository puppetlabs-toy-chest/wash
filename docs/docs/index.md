---
title: Docs
---
* [External plugins](external-plugins)
* [Analytics](#analytics)
  * [What data does Wash collect?](#what-data-does-wash-collect)
  * [Why does Wash collect data?](#why-does-wash-collect-data)
  * [How can I opt out of Wash data collection?](#how-can-i-opt-out-of-wash-data-collection)
* [Wash Commands](#wash-commands)
  * [wash](#wash)
  * [wash clear](#wash-clear)
  * [wash exec](#wash-exec)
  * [wash find](#wash-find)
  * [wash history](#wash-history)
  * [wash info](#wash-info)
  * [wash ls](#wash-ls)
  * [wash meta](#wash-meta)
  * [wash ps](#wash-ps)
  * [wash server](#wash-server)
  * [wash stree](#wash-stree)
  * [wash tail](#wash-tail)
  * [wash validate](#wash-validate)
  * [wash docs](#wash-docs)
  * [wash delete](#wash-delete)
  * [wash signal](#wash-signal)
* [Config](#config)
  * [wash.yaml](#wash-yaml)
  * [wash shell](#wash-shell)
* [CName](#cname)
* [Actions](#actions)
  * [list](#list)
  * [read](#read)
  * [write](#write)
  * [stream](#stream)
  * [exec](#exec)
  * [delete](#delete)
  * [signal](#signal)
* [Attributes](#attributes)
  * [crtime](#crtime)
  * [mtime](#mtime)
  * [ctime](#ctime)
  * [atime](#atime)
  * [size](#size)
  * [mode](#mode)
* [Server API](api)

## Analytics

Wash collects anonymous data about how you use it. You can opt out of providing this data.

### What data does Wash collect?
* Version of Wash
* User locale
* Architecture
* Method invocations (for shipped plugins only)
  * This includes any Wash action invocation
  * It also includes the entry's plugin

This data is associated with Bolt analytics' UUID (if available); otherwise, the data is associated with a random, non-identifiable user UUID.

### Why does Wash collect data?
Wash collects data to help us understand how it's being used and make decisions about how to improve it.

### How can I opt out of Wash data collection?
To disable the collection of analytics data add the following line to `~/.puppetlabs/wash/analytics.yaml`:

```
disabled: true
```

You can also disable the collection of analytics data by setting the `WASH_DISABLE_ANALYTICS` environment variable to `true` before starting up the Wash daemon.

## Wash Commands

Wash commands aim to be well-documented in the tool. Try `wash help` and `wash help <command>` for specific options.

Most commands operate on Wash resources, which are addressed by their path in the filesystem.

### wash

The `wash` command can be invoked on its own to enter a Wash shell.

Invoking `wash` starts the daemon as part of the process, then enters your current system shell with shortcuts configured for Wash commands. All the [`wash server`](#wash-server) settings are also supported with `wash` except `socket`; `wash` ignores that setting and creates a temporary location for the socket.

### wash clear

Wash caches most operations. If the resource you're querying appears out-of-date, use this subcommand to reset the cache for resources at or contained within the specified paths. Defaults to the current directory if no path is provided.

### wash exec

For a Wash resource that implements the ability to execute a command, run the specified command and arguments. The results will be forwarded from the target on stdout, stderr, and exit code.

### wash find

Recursively descends the directory tree of the specified paths, evaluating an `expression` composed of `primaries` and `operands` for each entry in the tree.

### wash history

Wash maintains a history of commands executed through it. Print that command history, or specify an `id` to print a log of activity related to a particular command.

Journals are stored in `wash/activity` under your user cache directory, identified by process ID and executable name. The user cache directory is `$XDG_CACHE_HOME` or `$HOME/.cache` on Unix systems, `$HOME/Library/Caches` on macOS, and `%LocalAppData%` on Windows.

### wash info

Prints the entries' info at the specified paths.

### wash ls

Lists the children of the specified paths, or current directory if no path is specified. If the `-l` option is set, then the name, last modified time, and supported actions are displayed for each child.

### wash meta

Prints the metadata of the given entries. By default, meta prints the full metadata as returned by the metadata endpoint. Specify the `--partial` flag to instead print the partial metadata, a (possibly) reduced set of metadata that's returned when entries are enumerated.

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

### wash docs

Displays the entry's documentation. This is currently its description and any supported signals/signal groups.

### wash delete

Deletes the entries at the specified paths, prompting the user for confirmation before deleting each entry.

### wash signal

Sends the specified signal to the entries at the specified paths.

## Config

### wash.yaml

The Wash config file is located at `~/.puppetlabs/wash/wash.yaml`, and can be used to configure the [`wash-server`](#wash-server). You can override this location via the `config-file` flag.

Below are all the configurable options.

* `logfile` - The location of the server's log file (default `stdout`)
* `loglevel` - The server's loglevel (default `info`)
* `cpuprofile` - The location that the server's CPU profile will be written to (optional)
* `external-plugins` - The external plugins that will be loaded. See [➠External Plugins]
* `plugins` - A list of shipped plugins to enable. If omitted or empty, it will load all of the shipped plugins. Note that Wash ships with the `docker`, `kubernetes`, `aws`, and `gcp` plugins.
* `socket` - The location of the server's socket file (default `<user_cache_dir>/wash/wash-api.sock`)

All options except for `external-plugins` can be overridden by setting the `WASH_<option>` environment variable with option converted to ALL CAPS.

NOTE: Do not override `socket` in a config file. Instead, override it via the `WASH_SOCKET` environment variable. Otherwise, Wash's commands will not be able to interact with the server because they cannot access the socket.

### wash shell

Wash uses your system shell to provide the shell environment. It determines this using the `SHELL` environment variable or falls back to `/bin/sh`, so if you'd like to specify a particular shell set the `SHELL` environment variable before starting Wash.

Wash uses the following environment variables

- `WASH_SOCKET` determines how to communicate with the Wash daemon
- `W` describes the path to Wash's starting directory on the host filesystem; use `cd $W` to return to the start or `ls $W/...` to list things relative to Wash's root
- `PATH` is prefixed with the location of the Wash binary and any other executables it creates

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
   3. Re-configure subcommand aliases, and configure the command prompt
   4. If `~/.washrc` exists, load it

That order ensures that the out-of-box experience of Wash is not adversely impacted by your existing environment while still inheriting most of your config. If you customize your Wash environment with `.washenv` and `.washrc`, be aware that it's possible to override Wash's default prompt and aliases.

For other shells, Wash creates executables for subcommands and does no other customization.

## CName

CName is short for _canonical name_. An entry's cname is its name with all `/`'es replaced by that entry's _slash replacer_. The default slash replacer is `#`.

Wash uses the cname to construct the entry's path. An entry's path is defined as
```
    <mountpoint>/<parent_cname1>/<parent_cname2>/.../<cname>
```

### Examples
Consider the entry `/myplugin/foo`. If `bar/baz` is a child of `foo`, then its cname and path would be `bar#baz` and `/myplugin/foo/bar#baz`, respectively. Similarly if `qux` is a child of `foo`, then its cname and path would be `qux` and `/myplugin/foo/qux`.

Conversely, if `bar/baz`'s slash replacer is set to `:`, then its cname and path would now be `bar:baz` and `/myplugin/foo/bar:baz`.

## Actions

### list
The `list` action lets you list an entry’s children. Entries that support list are represented as directories. Thus, any command that works with directories also work with these entries.

#### Examples
```
wash . ❯ ls gcp/Wash/storage/some-wash-stuff
an example folder reaper.sh
```

```
wash . ❯ cd gcp/Wash/storage/some-wash-stuff
wash gcp/Wash/storage/some-wash-stuff ❯
```

```
wash . ❯ tree gcp/Wash/storage/some-wash-stuff
gcp/Wash/storage/some-wash-stuff
├── an\ example\ folder
│   └── static.sh
└── reaper.sh

1 directory, 2 files
```

### read
The `read` action lets you read data from an entry. Thus, any command that reads a file also works with these entries.

#### Examples
```
wash . ❯ cat gcp/Wash/storage/some-wash-stuff/an\ example\ folder/static.sh
#!/bin/sh

echo "Hello, world!"
```

```
wash . ❯ grep "Hello" gcp/Wash/storage/some-wash-stuff/an\ example\ folder/static.sh
echo "Hello, world!"
```

### write
The `write` action lets you write data to an entry. Thus, any command that writes a file also works with these entries.

Note that Wash distinguishes between file-like and non-file-like entries. An entry is file-like if it's readable and writable and defines its size; you can edit it like a file.

If it doesn't define a size then it's non-file-like, and trying to open it with a ReadWrite handle will error; reads from it may not return data you previously wrote to it. You should check its documentation with the `docs` command for that entry's write semantics. We also recommend not using editors with these entries to avoid weird behavior.

#### Examples
Modifying a file stored in Google Cloud Storage
```
wash . ❯ echo 'exit 1' >> gcp/Wash/storage/some-wash-stuff/an\ example\ folder/static.sh
wash . ❯ cat gcp/Wash/storage/some-wash-stuff/an\ example\ folder/static.sh
#!/bin/sh

echo "Hello, world!"
exit 1
```

Writing a message to a hypothetical message queue where each write publishes a message and each read consumes a message
```
wash > echo 'message 1' >> myqueue
wash > echo 'message 2' >> myqueue
wash > cat myqueue
message 1
wash > cat myqueue
message 2
```

### stream
The `stream` action lets you stream an entry’s content for updates.

#### Examples
```
wash . ❯ tail -f gcp/Wash/compute/instance-1/fs/var/log/messages
===> gcp/Wash/compute/instance-1/fs/var/log/messages <===
Aug 27 06:25:01 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="17499" x-info="http://www.rsyslog.com"] rsyslogd was HUPed

Aug 27 13:26:32 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="17499" x-info="http://www.rsyslog.com"] exiting on signal 15.

Aug 27 13:26:32 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="24583" x-info="http://www.rsyslog.com"] start
Aug 28 00:30:04 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="24583" x-info="http://www.rsyslog.com"] exiting on signal 15.
Aug 28 00:30:04 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="32147" x-info="http://www.rsyslog.com"] start
Aug 28 06:25:01 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="32147" x-info="http://www.rsyslog.com"] rsyslogd was HUPed

Aug 28 09:54:34 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="32147" x-info="http://www.rsyslog.com"] exiting on signal 15.

Aug 28 09:54:34 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="6687" x-info="http://www.rsyslog.com"] start
Aug 28 19:01:21 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="6687" x-info="http://www.rsyslog.com"] exiting on signal 15.

Aug 28 19:01:21 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="12804" x-info="http://www.rsyslog.com"] start
```

(Hit `Ctrl+C` to cancel `tail -f`)

### exec
The `exec` action lets you execute a command on an entry.

#### Examples
```
wash . ❯ wexec gcp/Wash/compute/instance-1 uname
Linux
```

### delete
The `delete` action lets you delete an entry.

#### Examples
```
wash . ❯ delete docker/containers/quizzical_colden
remove docker/containers/quizzical_colden?: y
```

### signal
The `signal` action lets you signal an entry. Use the `docs` command to view an entry's supported signals.

#### Examples
```
wash . ❯ docs docker/containers/wash_tutorial_redis_1
No description provided.

SUPPORTED SIGNALS
* start
    Starts the container. Equivalent to 'docker start <container>'
* stop
    Stops the container. Equivalent to 'docker stop <container>'
* pause
    Suspends all processes in the container. Equivalent to 'docker pause <container>'
* resume
    Un-suspends all processes in the container. Equivalent to 'docker unpause <container>'
* restart
    Restarts the container. Equivalent to 'docker restart <container>'

SUPPORTED SIGNAL GROUPS
* linux
    Consists of all the supported Linux signals like SIGHUP, SIGKILL. Equivalent to
    'docker kill <container> --signal <signal>'
```

```
wash . ❯ signal start docker/containers/wash_tutorial_redis_1
wash . ❯
```

#### Common Signals
* start
* stop
* pause
* resume
* restart
* hibernate
* reset

## Attributes

### crtime
This is the entry's creation time.

#### Example JSON
As a stringified date

```
{
  "crtime": “2019-09-25T21:39:57-07:00”
}
```

In UNIX seconds

```
{
  "crtime": 1569472797
}
```

### mtime
This is the entry's last modification time.

#### Example JSON
As a stringified date

```
{
  "mtime": “2019-09-25T21:39:57-07:00”
}
```

In UNIX seconds

```
{
  "mtime": 1569472797
}
```

### ctime
This is the entry's last change time.

#### Example JSON
As a stringified date

```
{
  "ctime": “2019-09-25T21:39:57-07:00”
}
```

In UNIX seconds

```
{
  "ctime": 1569472797
}
```

### atime
This is the entry's last access time.

#### Example JSON
As a stringified date

```
{
  "atime": “2019-09-25T21:39:57-07:00”
}
```

In UNIX seconds

```
{
  "atime": 1569472797
}
```

### size
This is the entry's content size.

#### Example JSON
```
{
  "size": 1024
}
```

### mode
This is the entry's mode.

#### Example JSON

```
{
  "mode": 16832
}
```

As a hexadecimal string

```
{
  "mode": "41C0"
}
```

As an octal string

```
{
  "mode": "40700"
}
```
