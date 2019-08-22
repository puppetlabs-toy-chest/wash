+++
title= "Wash Tutorial"
+++

This tutorial assumes you've [installed Wash](../#getting-started).

## Setting up

To give you a sense of how `wash` works, we've created a multi-node Docker application based on the [Docker Compose tutorial](https://docs.docker.com/compose/gettingstarted). To start it run
```
svn export https://github.com/puppetlabs/wash.git/trunk/examples/swarm
docker-compose -f swarm/docker-compose.yml up -d
```

> If you don't have `svn` installed, you can `git clone https://github.com/puppetlabs/wash.git` and prefix `swarm/docker-compose.yml` with `wash/examples`.

This starts a small [Flask](http://flask.pocoo.org) webapp that keeps a count of how often it's been accessed in a [Redis](http://redis.io) instance that maintains state in a Docker volume.

> When done, run `docker-compose -f swarm/docker-compose.yml down` to stop the example application.

## Getting your bearings

Navigate the filesystem to view running containers
```
$ wash
Welcome to Wash!
  Wash includes several built-in commands: wexec, find, list, meta, tail.
  See commands run with wash via 'whistory', and logs with 'whistory <id>'.
Try 'help'
wash . ❯ cd docker/containers
wash docker/containers ❯ list
NAME             MODIFIED              ACTIONS
./               <unknown>             list
swarm_redis_1/   03 Jul 19 07:57 PDT   list, exec
swarm_web_1/     03 Jul 19 07:57 PDT   list, exec
wash docker/containers ❯ list swarm_web_1
NAME            MODIFIED              ACTIONS
./              03 Jul 19 07:57 PDT   list, exec
fs/             <unknown>             list
log             <unknown>             read, stream
metadata.json   <unknown>             read
```

Those containers are displayed as a directory, and provide access to their logs and metadata as files. Recent output from both can be accessed with common tools.
```
wash docker/containers ❯ tail */log
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
wash docker/containers ❯ meta swarm_web_1 -o yaml
AppArmorProfile: ""
Args:
- app.py
Config:
...
```

We can interrogate the container more closely with `wexec`
```
wash docker/containers ❯ wexec swarm_web_1 whoami
root
```

Try exploring `docker/volumes` to interact with the volume created for Redis.

## Finding with metadata

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

**NOTE:** `find \( -k 'docker/*container' -o -k 'kubernetes/*pod' \) -crtime -1d` is an easier (and more expressive) way to solve this problem. This example's only here to introduce the meta primary.

## Listing AWS resources

Wash also includes support for AWS. If you have your own and you've configured the AWS CLI on your workstation, you'll be able to use Wash to explore EC2 instances and S3 buckets.

As an example, you might want to periodically check how many execable instances are running in the AWS plugin (and display that via [BitBar](https://getbitbar.com/)):
```
running=`find aws -action exec -meta .State.Name running 2>/dev/null | wc -l | xargs`
total=`find aws -action exec -meta .State.Name -exists 2>/dev/null | wc -l | xargs`
echo EC2 $running / $total
```

Or count the number of S3 buckets that have been created:
```
buckets=`find aws -k '*s3*bucket' 2>/dev/null | wc -l | xargs`
echo S3 $buckets
```

## Record of activity

All operations have their activity recorded to journals. You can see a record of activity with `whistory`, and look at logs of individual entries with `whistory <id>`.

Journals are stored in `wash/activity` under your user cache directory, identified by process ID and executable name. The user cache directory is `$XDG_CACHE_HOME` or `$HOME/.cache` on Unix systems, `$HOME/Library/Caches` on macOS, and `%LocalAppData%` on Windows.
