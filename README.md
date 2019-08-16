# wash (Wide Area SHell)

[![GitHub release](https://img.shields.io/github/release/puppetlabs/wash.svg)](https://github.com/puppetlabs/wash/releases/) [![Build Status](https://travis-ci.com/puppetlabs/wash.svg)](https://travis-ci.com/puppetlabs/wash) [![GoDoc](https://godoc.org/github.com/puppetlabs/wash?status.svg)](https://godoc.org/github.com/puppetlabs/wash) [![Go Report Card](https://goreportcard.com/badge/github.com/puppetlabs/wash)](https://goreportcard.com/report/github.com/puppetlabs/wash)

`wash` helps you deal with all your remote or cloud-native infrastructure using the UNIX-y patterns and tools you already know and love!

For an introduction to Wash, see our main site at https://pup.pt/wash.

• [COMMUNITY](#community-feedback) • [DEVELOP](#developing-wash) • [ROADMAP](#roadmap)

## Community Feedback

We're actively soliciting community feedback and input on our [roadmap](#roadmap)! Don't hesitate to file issues for new features, new plugin types, new primitives, new command-line tools, or anything else that crosses your mind. You can also chat with us directly on [`#wash`](https://puppetcommunity.slack.com/app_redirect?channel=wash) on [Slack](https://slack.puppet.com/). Please abide by our [code of conduct](https://github.com/puppetlabs/wash/blob/master/CODE_OF_CONDUCT.md) when interacting with the community.

See the [roadmap](#roadmap) below to see what we've got planned!

We'd also love to get contributions from you! For a quick guide, take a look at our guide to [contributing](./CONTRIBUTING.md).

## Developing Wash

See https://pup.pt/wash/#getting-started for pre-requisites to run Wash.

Wash is a single binary application written in Go. It uses Go modules to identify dependencies.

To build it, run `go build`. To test, run `go test`.

> Requires golang 1.12+.

The latest pre-release version of the website can be found at https://puppetlabs.github.io/wash/dev.

## Roadmap

Project maintainers are not actively working on all of these things, but any of these are directions we would support others in pursuing.

### Primitives

* [ ] file/directory upload _(prereq for executing commands that aren't just one-liners)_
* [ ] edit a resource _(e.g. edit a file representing a k8s ConfigMap, and upon write save it via the k8s api)_
* [ ] delete a resource _(e.g. `rm`-ing a file in an S3 bucket deletes it)_
* [ ] signal handling to represent basic verbs _(e.g. sending a TERM to an EC2 instance will terminate it)_
* [ ] copy / move / rename _(how should this work?)_
* [ ] make `stream` able to "go back in time" _(e.g. support `tail -100 -f` style of "look-back")_

### Daemon enhancements

* [ ] rad startup ASCII art logo (<- high priority!)
* [X] expose plugin configuration via main config file
* [ ] expose what API calls are in-flight (to report status on large, distributed calls)

### CLI tools

* [ ] colorized output for `ls`, similar to `exa -l`
* [ ] make `ls` emit something useful when used against non-`wash` resources
* [ ] `exec` should work in parallel across multiple target resources
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

✓ = Implemented
○ = Possible, but not yet implemented
