# Contributing to Wash

## Code of Conduct

Review our [Code of Conduct](./CODE-OF-CONDUCT.md).

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

## Submitting Changes
Fork the repo, make changes, file a Pull Request.

### Guidelines

We try to follow common Go conventions as enforced by the compiler and several static analysis tools used in Travis CI.

#### File naming

File naming should follow camelCase. When grouping several files around a single concept, multiple names can be joined by hyphens.

For example, the Docker plugin has multiple files implenting components of a container. We name them
- container-log.go
- container-metadata.go
- container.go
