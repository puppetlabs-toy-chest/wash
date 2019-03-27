# wash (Wide Area SHell)

[![GitHub release](https://img.shields.io/github/release/puppetlabs/wash.svg)](https://github.com/puppetlabs/wash/releases/) [![Build Status](https://travis-ci.com/puppetlabs/wash.svg)](https://travis-ci.com/puppetlabs/wash) [![GoDoc](https://godoc.org/github.com/puppetlabs/wash?status.svg)](https://godoc.org/github.com/puppetlabs/wash) [![Go Report Card](https://goreportcard.com/badge/github.com/puppetlabs/wash)](https://goreportcard.com/report/github.com/puppetlabs/wash)

`wash` helps you deal with all your remote or cloud-native infrastructure using the UNIX-y patterns and tools you already know and love!

_TBD: Screencast_

Exploring, understanding, and inspecting modern infrastructure should be simple and straightforward. Whether it's containers, VMs, network devices, IoT stuff, or anything in between...they all have different ways of enumerating what you have, getting a stream of output, running commands, etc. Every vendor has its own tools and APIs that expose these features, each one different, each one bespoke. Thus, they are difficult to compose together to solve higher-level problems. And that's no fun at all!

[UNIX's philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Origin) and abstractions have worked for decades. They're pretty good, and more importantly, they're _familiar_ to millions of people. `wash` intends to apply those same philosophies and abstractions to modern, distributed infrastructure: With `wash`, we want to:

* make navigating stuff like servers, containers, or APIs as easy as navigating a local filesystem
* make scripting across your new-fangled infrastructure as easy as writing a local shell script
* render into text that which can be rendered into text (cuz text is a universal interface!) for easy viewing, editing, and UNIXy slicing-and-dicing
* build new versions of basic, UNIX tools to support the above goals (but reuse existing ones if they work!)

## Features

We've implemented some neat features inside of `wash` to support the above goals:

* The `wash` daemon
    * presents a FUSE filesystem hierarchy for all of your resources, letting you navigate them in normal, filesystem-y ways
    * preserves history of all executed commands, facilitating debugging
    * serves up an HTTP API for everything
    * caches information, for better performance

* Primitives - the basic building blocks that form the guts of `wash`, and dictate what kinds of things you can do to all the resources `wash` knows about
    * `list` - lets you ask any resource what's contained inside of it, and what primitives it supports. 
        - _e.g. listing a Kubernetes pod returns its constituent containers_
    * `metadata` - returns the metadata for any resource
        - _e.g. you can use this to get all the metadata for a docker container, or a file in an S3 bucket, etc._
    * `read` - lets you read the contents of a given resource
        - _e.g. represent an EC2 instance's console output as a regular file you can open in a regular editor_
    * `stream` - gives you streaming-read access to a resource
        - _e.g. to let you follow a container's output as its running_
    * `exec` - lets you execute a command against a resource
        - _e.g. run a shell command inside a container, or on an EC2 vm, or on a routerOS device, etc._

* CLI tools
    * `wash ls` - a version of `ls` that uses our API to enhance directory listings with `wash`-specific info
        - _e.g. show you what primitives are supported for each resource_
    * `wash meta` - uses the `metadata` primitive to emit a resource's metadata to standard out
    * `wash exec` - uses the `exec` primtitive to let you invoke commands against resources

* Core plugins (see the _Roadmap_ below for more details)
    * `docker` - presents a filesystem hierarchy of containers and volumes
        - found from the local socket or via `DOCKER` environment variables
    * `kubernetes` - presents a filesystem hierarchy of pods, containers, and persistent volume claims
        - uses contexts from `~/.kube/config`
    * `aws` - presents a filesystem hierarchy for EC2 and S3
        - uses `AWS_SHARED_CREDENTIALS_FILE` environment variable or `$HOME/.aws/credentials` and `AWS_CONFIG_FILE` environment variable or `$HOME/.aws/config` to find profiles and configure the SDK
        - IAM roles are supported when configured as described [here](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-role.html). Note that currently `region` will also need to be specified with the profile.
        - if using MFA, `wash` will prompt for it on standard input. Credentials are valid for 15 minutes. They are cached under `wash/aws-credentials` in your [user cache directory](#user-cache-directory) so they can be re-used across server restarts. `wash` may have to re-prompt for a new MFA token in response to navigating the `wash` environment to authorize a new session.

* [External plugins](https://github.com/puppetlabs/wash/tree/master/docs/external_plugins)
    * `wash` allows for easy creation of out-of-process plugins using any language you want, from `bash` to `go` or anything in-between!
    * `wash` handles the plugin lifecycle. it invokes your plugin with a certain calling convention; all you have to do is supply the business logic
    * users interact with external plugins the exact same way as core plugins; they are first-class citizens

## Installation

### Binaries

_TBD: See [GitHub releases](https://github.com/puppetlabs/wash/releases)._

### From Source

Clone repo and within it run `go install`.

Ensure `$GOPATH/bin` is part of `$PATH`.

> Requires golang 1.11+.

### Additional macOS Setup

> If using iTerm2, we recommend installing [iTerm2's shell integration](https://www.iterm2.com/documentation-shell-integration.html) to avoid [issue#84](https://github.com/puppetlabs/wash/issues/84).

Obtain FUSE for OSX [here](https://osxfuse.github.io/).

Add your mount directory to Spotlight's list of excluded directories to avoid heavy load.

## Usage

Mount `wash`'s filesystem and API server with
```
wash server mnt
```
In another shell, navigate to `mnt` to view available resources.

See available subcommands - such as `ls` and `exec` - with
```
wash help
```

When done, `cd` out of `mnt`, then run `umount mnt` or `Ctrl-C` the server process.

### Wash by Example

To get a sense of how `wash` works, we've included a multi-node Docker application based on the [Docker Compose tutorial](https://docs.docker.com/compose/gettingstarted). To start it run
```
docker-compose -f examples/swarm/docker-compose.yml up -d
```

> When done, run `docker-compose -f examples/swarm/docker-compose.yml down` to stop the example application.

This starts a small [Flask](http://flask.pocoo.org) webapp that keeps a count of how often it's been accessed in a [Redis](http://redis.io) instance that maintains state in a Docker volume.

Navigate the filesystem to view running containers
```
$ cd mnt/docker/containers
$ wash ls
NAME                                                                CREATED               ACTIONS
./                                                                  <unknown>             list
45a0265546d63a8f1b0d17033748db1468dc49dfd09cdaf2db62c45a60e82aaf/   20 Mar 19 17:02 PDT   exec, list, metadata
382776912d9373e6c4dc1350894b5290b22c36893a8fed08e2ba53fbb680c8a6/   20 Mar 19 17:02 PDT   exec, list, metadata
$ wash ls 382776912d9373e6c4dc1350894b5290b22c36893a8fed08e2ba53fbb680c8a6
NAME            CREATED               ACTIONS
./              20 Mar 19 17:02 PDT   exec, list, metadata
metadata.json   <unknown>             read
log             <unknown>             read, stream
```

Those containers are displayed as a directory, and provide access to their logs and metadata as files. Recent output from both can be accessed with common tools.
```
$ tail */log
==> 382776912d9373e6c4dc1350894b5290b22c36893a8fed08e2ba53fbb680c8a6/log <==
 * Serving Flask app "app" (lazy loading)
 * Environment: production
   WARNING: Do not use the development server in a production environment.
   Use a production WSGI server instead.
...

==> 45a0265546d63a8f1b0d17033748db1468dc49dfd09cdaf2db62c45a60e82aaf/log <==
1:C 21 Mar 2019 00:02:33.112 # oO0OoO0OoO0Oo Redis is starting oO0OoO0OoO0Oo
1:C 21 Mar 2019 00:02:33.112 # Redis version=5.0.4, bits=64, commit=00000000, modified=0, pid=1, just started
1:C 21 Mar 2019 00:02:33.112 # Configuration loaded
1:M 21 Mar 2019 00:02:33.113 * Running mode=standalone, port=6379.
...
```

The list earlier also noted that the container "directories" support the *metadata* action. We can get structured metadata in ether YAML or JSON with `wash meta`
```
$ wash meta 382776912d9373e6c4dc1350894b5290b22c36893a8fed08e2ba53fbb680c8a6 -o yaml
AppArmorProfile: ""
Args:
- app.py
Config:
...
```

We can interogate the container more closely with `wash exec`
```
$ wash exec 45a0265546d63a8f1b0d17033748db1468dc49dfd09cdaf2db62c45a60e82aaf whoami
root
```

Try exploring `mnt/docker/volumes` to interact with the volume created for Redis.

### Record of Activity

All operations have their activity recorded to journals in `wash/activity` under your [user cache directory](#user-cache-directory), identified by process ID and executable name.

### User Cache Directory

`wash` uses a user-specific cache directory to store running state. The user cache directory is `$XDG_CACHE_HOME` or `$HOME/.cache` on Unix systems, `$HOME/Library/Caches` on macOS, and `%LocalAppData%` on Windows.

## Known Issues

### On macOS

If the `wash` daemon exits with a exit status of 255, that typically means that `wash` couldn't load the FUSE extensions. MacOS only allows for a certain (small) number of virtual devices on the system, and if all available slots are taken up by other programs then we won't be able to run. You can view loaded extensions with `kextstat`. More information in [this github issue for *FUSE for macOS*](https://github.com/osxfuse/osxfuse/issues/358).

## Roadmap

We're activilely soliciting community feedback and input on our roadmap! Don't hesitate to file issues for new features, new plugin types, new primitives, new command-line tools, or anything else that crosses your mind. You can also chat with us directly on `#wash` on [Slack](https://slack.puppet.com/).

### Primitives

* [ ] file/directory upload _(prereq for executing commands that aren't just one-liners)_
* [ ] edit a resource _(e.g. edit a file representing a k8s ConfigMap, and upon write save it via the k8s api)_
* [ ] delete a resource _(e.g. `rm`-ing a file in an S3 bucket deletes it)_
* [ ] signal handling to represent basic verbs _(e.g. sending a TERM to an EC2 instance will terminate it)_
* [ ] copy / move / rename _(how should this work?)_
* [ ] make `stream` able to "go back in time" _(e.g. support `tail -100 -f` style of "look-back")_

### Daemon enhancements

* [ ] rad startup ASCII art logo (<- high priority!)
* [ ] expose plugin configuration via main config file
* [ ] expose what API calls are in-flight (to report status on large, distributed calls)

### CLI tools

* [ ] colorized output for `ls`, similar to `exa -l`
* [ ] make `ls` emit something useful when used against non-`wash` resources
* [ ] `tail` that works for `wash` resources that support `stream`
* [ ] `exec` should work in parallel across multiple target resources
* [ ] `history` that lets you explore `wash`'s log/journal
* [ ] `find` that lets you refer to `wash` primitives _(e.g. find all the resources under `/docker` that support `exec`)_
* [ ] build an interactive shell that works over `exec` _(need to update plugins API to support this, most likely)_
* [ ] a version of `top` that works using `wash` primitives to get information to display from multiple targets

### Plugins / content

|   | `list` | `read` | `stream` | `exec` | `meta` |
| - | :-: | :-: | :-: | :-: | :-: |
| **Docker** |
| Containers | ✓ | | | ✓ | ✓ |
| Container logs | | ✓ | ✓ |
| Volumes | ✓ | ✓ | ○ | | ✓ |
| Images | ○ | | | | ○ |
| Networks | ○ | | | | ○ |
| Services | ○ | ○ | ○ | | ○ |
| Stacks | ○ | | | | ○ |
| Swarm nodes | ○ | | | | ○ |
| Swarm config | ○ | ○ | | | ○ |
| **Kubernetes** |
| Pods | ✓ | ✓ | ✓ | ✓ | ✓ |
| Persistent Volume Claims | ✓ | ✓ | ✓ | | ✓ |
| Services | ○ | | | | ○ |
| ConfigMaps | ○ | ○ | | | ○ |
| _generic k8s resources_ | ○ | | | | ○ |
| **AWS** |
| EC2 | ✓ | ✓ | ○ | ✓ | ✓ |
| S3 buckets | ✓ | | | | ✓ |
| S3 directories | ✓ |
| S3 objects | | ✓ | ✓ | | ✓ |
| Cloudwatch | ○ | ○ | ○ | | ○ |
| Lambda | ○ | ○ | ○ | ○ | ○ |
| _pubsub (e.g. SNS)_ | ○ | | ○ | | ○ |
| _databases (e.g. dynamo, RDS)_ | ○ | ○ | ○ | ○ | ○ |
| _networking (e.g. ELB, Route53)_ | ○ | ○ | ○ | ○ | ○ |
| **SSH/WinRM targets** | ○ | | | ○ | |
| **SSHfs** | ○ | ○ | ○ | | |
| **GCP** | ○ | ○ | ○ | ○ | ○ |
| **Azure** | ○ | ○ | ○ | ○ | ○ |
| **VMware** | ○ | ○ | ○ | ○ | ○ |
| **Splunk** | | ○ | ○ | ○ | |
| **Logstash** | | ○ | ○ | ○ | |
| **_Network Devices (e.g. Cisco)_** | ○ | ○ | ○ | ○ | ○ |
| **_IoT (e.g. Nest, Hue, Rachio)_** | ○ | ○ | ○ | ○ | ○ |
| **`wash` itself (expose internals)** | ○ | ○ | ○ | ○ | ○ |

## Contributing

We'd love to get contributions from you! For a quick guide, take a look at our guide to [contributing](./CONTRIBUTING.md).
