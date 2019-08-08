+++
title= "wash: the wide-area shell"
date= 2019-04-19T22:59:26-06:00
description = ""
draft= false
+++

Wash helps you deal with all your remote or cloud-native infrastructure using the UNIX-y patterns and tools you already know and love.

<div id="horizontalmenu">
    • <a href="#introduction">introduction</a>
    • <a href="#getting-started">getting started</a>
    • <a href="#wash-by-example">wash by example</a>
    • <a href="#current-features">features</a>
    • <a href="#contributing">contributing</a>
    •
</div>

<script id="asciicast-pTl39ujtBBkIYOZMZSWuE8WRN" src="https://asciinema.org/a/pTl39ujtBBkIYOZMZSWuE8WRN.js" async></script>

## Introduction

Exploring, understanding, and inspecting modern infrastructure should be simple and straightforward. Whether it's containers, VMs, network devices, IoT stuff, or anything in between...they all have different ways of enumerating what you have, getting a stream of output, running commands, etc. Every vendor has its own tools and APIs that expose these features, each one different, each one bespoke. Thus, they are difficult to compose together to solve higher-level problems. And that's no fun at all!

[UNIX's philosophy](https://en.wikipedia.org/wiki/Unix_philosophy#Origin) and abstractions have worked for decades. They're pretty good, and more importantly, they're _familiar_ to millions of people. Wash intends to apply those same philosophies and abstractions to modern, distributed infrastructure.

Wash aims to:

* help you keep track of what systems you're running
* make scripting across your new-fangled infrastructure as easy as writing a local shell script
* provide easy access to files, logs, metadata, and any other activity for viewing, editing, and UNIXy slicing-and-dicing
* help you build better tools that work across a variety of cloud APIs
* let you easily build your own integrations in whatever language you want
* be extremely simple to get up-and-running; if it takes you more than a few minutes, let us know!

## Getting started

Wash is distributed as a single binary, and the only prerequisite is [`libfuse`](https://github.com/libfuse/libfuse). Thus, getting going is pretty simple:

1. [Download](https://github.com/puppetlabs/wash/releases) the Wash binary for your platform
   * or install with `brew install puppetlabs/puppet/wash`
2. Install `libfuse`, if you haven't already
   * E.g. on MacOS using homebrew: `brew cask install osxfuse`
   * E.g. on CentOS: `yum install fuse fuse-libs`
   * E.g. on Ubuntu: `apt-get install fuse`
3. Start Wash
   * `./wash`

At this point, if you haven't already, you should start some resources that Wash can actually introspect. Otherwise, as Han Solo would say, "this is going to be a real short trip". So fire up some Docker containers, create some EC2 instances, toss some files into S3, launch a Kubernetes pod, etc. We've also provided a [guided tour](#wash-by-example) that includes a Docker application.

**NOTE:** Wash collects anonymous data about how you use it. See the [analytics docs](/wash/docs#analytics) for more details.

### Known issues

Wash uses your system shell to provide the shell environment. It determines this using the SHELL environment variable or falls back to `/bin/sh`, so if you'd like to specify a particular shell set the SHELL environment variable before starting Wash.

#### On macOS

If using iTerm2, we recommend installing [iTerm2's shell integration](https://www.iterm2.com/documentation-shell-integration.html) to avoid [issue#84](https://github.com/puppetlabs/wash/issues/84).

If the `wash` daemon exits with a exit status of 255, that typically means that `wash` couldn't load the FUSE extensions. MacOS only allows for a certain (small) number of virtual devices on the system, and if all available slots are taken up by other programs then we won't be able to run. You can view loaded extensions with `kextstat`. More information in [this github issue for *FUSE for macOS*](https://github.com/osxfuse/osxfuse/issues/358).

## Wash by example

To give you a sense of how `wash` works, we've created a multi-node Docker application based on the [Docker Compose tutorial](https://docs.docker.com/compose/gettingstarted). To start it run
```
svn export https://github.com/puppetlabs/wash.git/trunk/examples/swarm
docker-compose -f swarm/docker-compose.yml up -d
```

> If you don't have `svn` installed, you can `git clone https://github.com/puppetlabs/wash.git` and prefix `swarm/docker-compose.yml` with `wash/examples`.

This starts a small [Flask](http://flask.pocoo.org) webapp that keeps a count of how often it's been accessed in a [Redis](http://redis.io) instance that maintains state in a Docker volume.

> When done, run `docker-compose -f swarm/docker-compose.yml down` to stop the example application.

Navigate the filesystem to view running containers
```
$ wash
Welcome to Wash!
  Wash includes several built-in commands: wexec, find, list, meta, tail.
  See commands run with wash via 'whistory', and logs with 'whistory <id>'.
Try 'help'
wash$ cd docker/containers
wash$ list
NAME             MODIFIED              ACTIONS
./               <unknown>             list
swarm_redis_1/   03 Jul 19 07:57 PDT   list, exec
swarm_web_1/     03 Jul 19 07:57 PDT   list, exec
wash$ list swarm_web_1
NAME            MODIFIED              ACTIONS
./              03 Jul 19 07:57 PDT   list, exec
fs/             <unknown>             list
log             <unknown>             read, stream
metadata.json   <unknown>             read
```

Those containers are displayed as a directory, and provide access to their logs and metadata as files. Recent output from both can be accessed with common tools.
```
wash$ tail */log
==> swarm_web_1/log <==
 * Serving Flask app "app" (lazy loading)
 * Environment: production
   WARNING: Do not use the development server in a production environment.
   Use a production WSGI server instead.
...

==> swarm_redis_1/log <==
1:C 21 Mar 2019 00:02:33.112 # oO0OoO0OoO0Oo Redis is starting oO0OoO0OoO0Oo
1:C 21 Mar 2019 00:02:33.112 # Redis version=5.0.4, bits=64, commit=00000000, modified=0, pid=1, just started
1:C 21 Mar 2019 00:02:33.112 # Configuration loaded
1:M 21 Mar 2019 00:02:33.113 * Running mode=standalone, port=6379.
...
```

Notice that tab-completion makes it easy to find the containers you want to explore.

The list earlier also noted that the container "directories" support the *metadata* action. We can get structured metadata in ether YAML or JSON with `wash meta`
```
wash$ meta swarm_web_1 -o yaml
AppArmorProfile: ""
Args:
- app.py
Config:
...
```

We can interrogate the container more closely with `wexec`
```
wash$ wexec swarm_web_1 whoami
root
```

Try exploring `docker/volumes` to interact with the volume created for Redis.

### Finding with metadata

Wash includes a powerful `find` command that can match based on an entry's metadata. For example, if we wanted to see what containers were created recently, we would look at the `.Created` entry for Docker containers and the `.metadata.creationTimestamp` for Kubernetes pods. We can find all instances created in the last 24 hours with

```
find -meta .Created -24h -o -meta .metadata.creationTimestamp -24h
```

That says: list entries with the `Created` metadata - interpreted as a date-time - that falls within the last 24 hours, or that have the `metadata: creationTimestamp` in the last 24 hours. See `help find` for more on using `find`.

If you want to try this out, or explore more Kubernetes functionality, you can create a Redis server with a persistent volume using Kubernetes in Docker and the following config:

```
cat <<EOF | kubectl create -f -
kind: PersistentVolume
apiVersion: v1
metadata:
  name: redis
  labels:
    type: local
spec:
  storageClassName: manual
  capacity:
    storage: 100Mi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/mnt/data"
---
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: redis
spec:
  storageClassName: manual
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Mi
---
kind: Pod
apiVersion: v1
metadata:
  name: redis
spec:
  containers:
  - name: redis
    image: redis
    args: ["--appendonly", "yes"]
    volumeMounts:
    - name: redis
      mountPath: /data
  volumes:
  - name: redis
    persistentVolumeClaim:
      claimName: redis
EOF
```

### Listing AWS resources

Wash also includes support for AWS. If you have your own and you've configured the AWS CLI on your workstation, you'll be able to use Wash to explore EC2 instances and S3 buckets.

As an example, you might want to periodically check how many EC2 instances are running (and display that via [BitBar](https://getbitbar.com/)):
```
running=`find aws -action exec -meta .State.Name running 2>/dev/null | wc -l | xargs`
total=`find aws -action exec -meta .State.Name -exists 2>/dev/null | wc -l | xargs`
echo EC2 $running / $total
```

Or count the number of S3 buckets that have been created:
```
buckets=`find aws -maxdepth 4 -path '*/resources/s3/*' 2>/dev/null | wc -l | xargs`
echo S3 $buckets
```

### Record of activity

All operations have their activity recorded to journals. You can see a record of activity with `whistory`, and look at logs of individual entries with `whistory <id>`.

Journals are stored in `wash/activity` under your user cache directory, identified by process ID and executable name. The user cache directory is `$XDG_CACHE_HOME` or `$HOME/.cache` on Unix systems, `$HOME/Library/Caches` on macOS, and `%LocalAppData%` on Windows.

## Current features

Wash does a lot already, with [more to come](https://github.com/puppetlabs/wash#roadmap):

* presents a filesystem hierarchy for all of your resources, letting you navigate them in normal, filesystem-y ways
* preserves history of all executed commands, facilitating debugging
* serves up an HTTP API for everything
* caches information, for better performance

We've implemented a number of handy Wash commands ([docs](/wash/docs#wash-commands)):

* `wash ls` - a version of `ls` that uses our API to enhance directory listings with Wash-specific info
  - _e.g. show you what primitives are supported for each resource_
* `wash meta` - emits a resource's metadata to standard out
* `wash exec` - uses the `exec` primitive to let you invoke commands against resources
* `wash find` - find resources using powerful selection predicates
* `wash tail -f` - follow updates to resources that support the `stream` primitive as well as normal files
* `wash ps` - lists running processes on indicated compute instances that support the `exec` primitive
* `wash history` - lists all activity through Wash; `wash history <id>` can be used to view logs for a specific activity
* `wash clear` - clears cached data for a sub-hierarchy rooted at the supplied path so Wash will re-request it

[Core plugins](/wash/docs#core-plugins) (and we're [adding more all the time](https://github.com/puppetlabs/wash#roadmap), see our [docs](/wash/docs#plugin-concepts) for how to help):

* [docker](/wash/docs#docker): containers and volumes
* [kubernetes](/wash/docs#kubernetes): pods, containers, and persistent volume claims
* [aws](/wash/docs#aws): EC2 and S3
* [gcp](/wash/docs#gcp): Compute Engine and Storage

[External plugins](/wash/docs/external_plugins):

* Wash allows for easy creation of out-of-process plugins using any language you want, from `bash` to `go` or anything in-between!
* Wash handles the plugin life-cycle. it invokes your plugin with a certain calling convention; all you have to do is supply the business logic
* users interact with external plugins the exact same way as core plugins; they are first-class citizens

Several external plugins have already been created:

* [Washhub](https://github.com/timidri/washhub) - navigate all your GitHub repositories at once
* [Washreads](https://github.com/MikaelSmith/washreads) - view your Goodreads bookshelves; also structured as an introduction to writing external plugins
* [Puppetwash](https://github.com/timidri/puppetwash) - view your Puppet (Enterprise) instances and information about the managed nodes

If you've created an external plugin, please put up a pull request to add it to this list!

For more information about future direction, see our [Roadmap](https://github.com/puppetlabs/wash#roadmap)!

## Contributing

There are tons of ways to get involved with Wash, whether or not you're a programmer!

- Come and hang out with us on [Slack](https://puppetcommunity.slack.com/app_redirect?channel=wash)! Feel free to ask questions, answer questions from other folks, or just lurk. Come and talk to us about things about modern infra you find [complex or infuriating](https://landscape.cncf.io/), or what your [favorite hacking movie scenes](https://www.youtube.com/watch?v=u1Ds9CeG-VY) are, or the [best monospaced font](https://fonts.google.com/specimen/Inconsolata). 

- Have something cool that you'd like connect up to Wash? We'd love to hear your ideas, and help you figure out how to do it! We don't care if you want Wash to talk to a network device, a smart lightbulb, your bluetooth-enabled espresso scale, or just more types of resources from cloud providers. 

- Are you an artist? Design some Wash-related artwork or a logo, and we'll see about putting it into the rotation for the site!

- Are you an old skool command-line gearhead with, like, *opinions* about how things should work on a command line? We'd love your help figuring out how the shell experience for Wash should work. How can our unixy Wash commands behave better? Are there new commands we should build? What colors and formatting should we use for `wash ls`? If we implemented `wash fortune`, what quotes should be in there?!

- Did you script something cool that uses Wash under the hood? Please let us know, and how we can help!

- Can you sling HTML, or Markdown? This site is built using Hugo, and the source is in our [github repo](https://github.com/puppetlabs/wash/tree/master/website). We'd love help documenting stuff!

- Did you give a talk or presentation on Wash? Give us the link, so we can help promote it!

Come check us out on [github](https://github.com/puppetlabs/wash), and in particular check out the [contribution guidelines](https://github.com/puppetlabs/wash/blob/master/CONTRIBUTING.md) and the [code of conduct](https://github.com/puppetlabs/wash/blob/master/CODE_OF_CONDUCT.md).
