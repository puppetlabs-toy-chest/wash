---
title: Commands
---

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

Wash commands aim to be well-documented in the tool. Try `wash help` and `wash help <command>` for specific options.

Most commands operate on Wash resources, which are addressed by their path in the filesystem.

## wash

The `wash` command can be invoked on its own to enter a Wash shell.

Invoking `wash` starts the daemon as part of the process, then enters your current system shell with shortcuts configured for Wash commands. All the [`wash server`](#wash-server) settings are also supported with `wash` except `socket`; `wash` ignores that setting and creates a temporary location for the socket.

## wash clear

Wash caches most operations. If the resource you're querying appears out-of-date, use this subcommand to reset the cache for resources at or contained within the specified paths. Defaults to the current directory if no path is provided.

## wash exec

For a Wash resource that implements the ability to execute a command, run the specified command and arguments. The results will be forwarded from the target on stdout, stderr, and exit code.

## wash find

Recursively descends the directory tree of the specified paths, evaluating an `expression` composed of `primaries` and `operands` for each entry in the tree.

## wash history

Wash maintains a history of commands executed through it. Print that command history, or specify an `id` to print a log of activity related to a particular command.

Journals are stored in `wash/activity` under your user cache directory, identified by process ID and executable name. The user cache directory is `$XDG_CACHE_HOME` or `$HOME/.cache` on Unix systems, `$HOME/Library/Caches` on macOS, and `%LocalAppData%` on Windows.

## wash info

Prints the entries' info at the specified paths.

## wash ls

Lists the children of the specified paths, or current directory if no path is specified. If the `-l` option is set, then the name, last modified time, and supported actions are displayed for each child.

## wash meta

Prints the metadata of the given entries. By default, meta prints the full metadata as returned by the metadata endpoint. Specify the `--partial` flag to instead print the partial metadata, a (possibly) reduced set of metadata that's returned when entries are enumerated.

## wash ps

Captures /proc/*/{cmdline,stat,statm} on each node by executing 'cat' on them. Collects the output
to display running processes on all listed nodes. Errors on paths that don't implement exec.

## wash server

Initializes all of the plugins, then sets up the Wash daemon (its API and [FUSE](https://en.wikipedia.org/wiki/Filesystem_in_Userspace) servers). To stop it, make sure you're not using the filesystem at the specified mountpoint, then enter Ctrl-C.

Server API docs can be found [here](api). The server config is described in the [`config`](#config) section.

## wash stree

Displays the entry's stree (schema-tree), which is a high-level overview of the entry's hierarchy. Non-singleton types are bracketed with "[]".

## wash tail

Output any new updates to files and/or resources (that support the stream action). Currently requires the '-f' option to run. Attempts to mimic the functionality of `tail -f` for remote logs.

## wash validate

Validates an external plugin, using it's schema to limit exploration. The plugin can be one you've configured in Wash's config file, or it can be a script to load as an external plugin. Plugin-specific config from Wash's config file will be used. The Wash daemon does not need to be running to use this command.

Validate starts from the plugin root and does a breadth-first traversal of the plugin hierarchy, invoking all supported methods on an example at each level. If the plugin provides a schema, it will be used to explore one example of each type of entry. Exploration can be stopped with Ctrl-C when needed.

Each line represents validation of an entry type. The `lrsx` fields represent support for `list`, `read`, `stream`, and `execute` methods respectively, with '-' representing lack of support for a method.

## wash docs

Displays the entry's documentation. This is currently its description and any supported signals/signal groups.

## wash delete

Deletes the entries at the specified paths, prompting the user for confirmation before deleting each entry.

## wash signal

Sends the specified signal to the entries at the specified paths.
