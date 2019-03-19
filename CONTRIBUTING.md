# Contributing to Wash

## Code of Conduct

Review our [Code of Conduct](./CODE_OF_CONDUCT.md).

## How to ask a question

Browse for existing issues. Otherwise, open a "[new issue](https://github.com/puppetlabs/wash/issues/new)" in this repo.

## How to report a bug

Open a "[Bug report](https://github.com/puppetlabs/wash/issues/new?template=bug_report.md)" issue in this repo.

## How to suggest a new feature

Open a "[Feature request](https://github.com/puppetlabs/wash/issues/new?template=feature_request.md)" issue in this repo.

## Development Environment

### Requirements

* Golang 1.11

### Building

You can build a native binary with `go build`. The resulting `wash` binary will be placed in the current directory.

## How to create a new plugin

The [plugin](./plugin) package defines a set of interfaces that a plugin can implement to enable specific behaviors.

### Starting a plugin

Create a new directory in [plugin](./plugin) for the plugin.

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
- Files and directories that exist on a remote system or volume should implement the `File` interface. Additionally the [volume](./volume) package provides helpers for representing these types.
- To support executing commands on a resource, implement the `Execable` interface.

Tip: base new resources on the `EntryBase` struct to simplify creating them and enable caching.

## Submitting Changes
Fork the repo, make changes, file a Pull Request.

Contributions to this project require sign-off consistent with the [Developers Certificate of Origin](https://developercertificate.org). This can be as simple as using `git commit -s` on each commit.

### Guidelines

We try to follow common Go conventions as enforced by the compiler and several static analysis tools used in Travis CI.

#### File naming

File naming should follow camelCase. When grouping several files around a single concept, multiple names can be joined by hyphens.

For example, the Docker plugin has multiple files implenting components of a container. We name them
- container-log.go
- container-metadata.go
- container.go
