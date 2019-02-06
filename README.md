# wash (Wide Area SHell)

A cloud-native shell for bringing remote infrastructure to your terminal.

## Usage

This prototype is built as a FUSE filesystem. It currently only supports viewing running containers in Docker (found from the local socket or via DOCKER environment variables) and Kubernetes (uses the currently selected context from `~/.kube/config`)

Mount the filesystem with
```
go run wash.go mnt
```

In another shell navigate it at `mnt`. When done `umount mnt`.

Operations that work:
- `ls`
- `cat`
- `vim`
- `tail [-f]`
- `stat` (kind of, information's not very useful)
- `xattr` and a new command `meta`

NOTE: requires golang 1.11.4.

Obtain FUSE for OSX [here](https://osxfuse.github.io/).

### Container

You can also use it as a container.

Build the container with
```
docker build . -t wash
```

Run with
```
docker run --rm --name wash --device /dev/fuse --cap-add SYS_ADMIN wash
```

If you want to be able to access Docker instances from your local Docker runtime add
```
-v /var/run/docker.sock:/var/run/docker.sock
```

To pull in your local config add `-v $HOME:/root`. Additionally add `-v $HOME:$HOME` if you use symlinks with any of your config.

Then start a shell and explore with
```
docker exec -it wash sh
```

## Principles
- Multiple ways to get data, but consistent language within the tool. i.e. may search for a database by saying type is 'db' or 'database', but the tool will always refer to them by 'database'.
- Rich shell experience.
- Store everything for future use.

## Examples

> This is a collection of examples showing how we think `wash` could work. The actual project doesn't currently reflect many of these patterns.

We approach the shell as a way of understanding what cloud resources exist that we can see, and a set of tools for interacting with them.

### Navigation

Basic shell interaction is around navigating through a hierarchy and understanding what's there. For that we inherit some classic shell commands: `ls`, `cd`, `tree`, `pwd`.

The hierarchy created first distinguishes between the cloud environments containing resources, then several ways of grouping those resources. It shows all configured APIs, where we ideally pickup confguration from the location their own CLIs use (config files, environment variables).

```fish
> ls /
gcp/
aws/
platform9/
kubernetes/
> cd aws/
> ls
groups/
regions/
resources/
```

We equate resources appearing in multiple groupings as hardlinks in a filesystem; there are several ways to get to the same resource. Specific APIs may have different ways of grouping resources, such as groups and tags in AWS or namespace in Kubernetes.

```fish
> pwd
/aws
> tree groups/
groups
├── developer
│   ├── ec2
│   │   ├── vm-106.puppet.com
│   │   └── vm-107.puppet.com
│   ├── lambda
│   │   └── michael-lambda-17
│   └── s3
│       └── michael-bucket1
│           └── <files in the bucket>
└── dujour
    ├── ec2
    │   ├── vm-106.puppet.com
    │   └── vm-107.puppet.com
    ├── lambda
    │   └── michael-lambda-17
    └── s3
        └── michael-bucket1
            └── <files in the bucket>
```

Cloud vendors have many (many) types of resources. We only show the ones you actually use.

```fish
> cd /aws/resources/
> ls					            # Only show types of things where you have resources
ec2
lambda
s3
> cd ec2/
> ls -l                             # Show some details about ownership and categorization
Name                Creator         Groups              Created         Tags
vm-106.puppet.com   michael.smith   developer,dujour    Dec 29 10:41    prod,web
```

We aim to initially support a small set of cloud environments, such as AWS and Kubernetes, and enable a community of folks to expand that to additional environments.

```fish
> ls /kubernetes/
docker-for-desktop/
gke_shared-k8s_us-west1-a_shared-k8s-dev/
gke_shared-k8s_us-west1-a_shared-k8s-prod/
gke_shared-k8s_us-west1-a_shared-k8s-stage/
```

Like any modern shell, history is saved.

```fish
> history
 1  1/6/2019 10:15  ls /
 2  1/6/2019 10:16  cd aws/
 3  1/6/2019 10:16  ls
 4  1/6/2019 10:17  pwd
 5  1/6/2019 10:17  tree groups/
 6  1/6/2019 10:17  cd /aws/resources/
 7  1/6/2019 10:18  ls
 8  1/6/2019 10:18  cd ec2/
 9  1/6/2019 10:20  ls -l
10  1/6/2019 10:20  ls /kubernetes/
```

Not only the commands are saved. All data used to obtain the result, as well as a structured version of the result, is stored and accessible. See [troubleshooting](#troubleshooting) for more.

### Understanding

Higher level operations in a shell might involve searching for types of things and finding out some information about them.

We aim to create or adopt a taxonomy around resources across cloud environments to enable searching for things that are alike. Note that we only show the shortest path to each resource.

```fish
> find / -type compute,storage
aws/resources/ec2/vm-106.puppet.com
aws/resources/ec2/vm-107.puppet.com
aws/resources/s3/michael-bucket1
kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/pods/r0raxmg1fg276o05wmmqancki8w-dujour-84c7b497cc-fd7m4
```

Once we've found some resources, we'd like to see what they're doing. The default output for a VM is its standard log (stdout for containers or lamdas, /var/log/syslog, /var/log/messages, Windows Event Viewer, Mac System Log, access logs for storage when enabled), but we should be able to access other logs on the system as well.

```fish
> tail -f /aws/resources/ec2/vm-106.puppet.com /aws/resources/ec2/vm-107.puppet.com:/var/log/nginx/access.log /aws/resources/lambda/michael-lambda-17 /aws/resources/s3/michael-bucket1 /kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/pods/r0raxmg1fg276o05wmmqancki8w-dujour-84c7b497cc-fd7m4
==> /aws/resources/ec2/vm-107.puppet.com:/var/log/syslog <==
Jan  2 23:53:50 pe-master systemd[1]: Starting User Manager for UID 1000...
Jan  2 23:53:50 pe-master systemd[1]: Started Session 25386 of user ubuntu.

==> /aws/resources/ec2/vm-107.puppet.com:/var/log/nginx/access.log <==
10.0.25.192 - - [20/Dec/2018:18:03:58 +0000] "GET /index.html HTTP/1.1" 200 603 "https://vm-107.puppet.com/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0.1 Safari/605.1.15" "-"
10.0.25.192 - - [20/Dec/2018:18:04:59 +0000] "GET /index.html HTTP/1.1" 200 605 "https://vm-107.puppet.com/" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0.1 Safari/605.1.15" "-"

==> /aws/resources/lambda/michael-lambda-17 <==
=== puma startup: 2019-01-02 06:25:01 +0000 ===

==> /aws/resources/s3/michael-bucket1
79a59df900b949e55d96a1e698fbacedfd6e09d98eacf8f8d5218e7cd47ef2be mybucket [06/Feb/2014:00:00:38 +0000] 192.0.2.3 79a59df900b949e55d96a1e698fbacedfd6e09d98eacf8f8d5218e7cd47ef2be 3E57427F3EXAMPLE REST.GET.VERSIONING - "GET /mybucket?versioning HTTP/1.1" 200 - 113 - 7 - "-" "S3Console/0.4" -
79a59df900b949e55d96a1e698fbacedfd6e09d98eacf8f8d5218e7cd47ef2be mybucket [06/Feb/2014:00:00:38 +0000] 192.0.2.3 79a59df900b949e55d96a1e698fbacedfd6e09d98eacf8f8d5218e7cd47ef2be 891CE47D2EXAMPLE REST.GET.LOGGING_STATUS - "GET /mybucket?logging HTTP/1.1" 200 - 242 - 11 - "-" "S3Console/0.4" -
79a59df900b949e55d96a1e698fbacedfd6e09d98eacf8f8d5218e7cd47ef2be mybucket [06/Feb/2014:00:00:38 +0000] 192.0.2.3 79a59df900b949e55d96a1e698fbacedfd6e09d98eacf8f8d5218e7cd47ef2be A1206F460EXAMPLE REST.GET.BUCKETPOLICY - "GET /mybucket?policy HTTP/1.1" 404 NoSuchBucketPolicy 297 - 38 - "-" "S3Console/0.4" -
79a59df900b949e55d96a1e698fbacedfd6e09d98eacf8f8d5218e7cd47ef2be mybucket [06/Feb/2014:00:01:00 +0000] 192.0.2.3 79a59df900b949e55d96a1e698fbacedfd6e09d98eacf8f8d5218e7cd47ef2be 7B4A0FABBEXAMPLE REST.GET.VERSIONING - "GET /mybucket?versioning HTTP/1.1" 200 - 113 - 33 - "-" "S3Console/0.4" -

==> /kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/pods/r0raxmg1fg276o05wmmqancki8w-dujour-84c7b497cc-fd7m4 <==
2018-12-06 19:23:53,215 INFO  [o.e.j.u.log] Logging initialized @49306ms to org.eclipse.jetty.util.log.Slf4jLog
2018-12-06 19:23:54,335 INFO  [p.t.s.w.jetty9-core] Removing buggy security provider SunPKCS11 version 12
2018-12-06 19:23:58,485 INFO  [p.t.s.w.jetty9-service] Initializing web server(s).
2018-12-06 19:23:58,525 INFO  [p.t.s.s.scheduler-service] Initializing Scheduler Service
2018-12-06 19:23:58,635 INFO  [o.q.i.StdSchedulerFactory] Using default implementation for ThreadExecutor

```

Metadata on specific VMs can be accessed via `stat`.

```fish
> stat /kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/pods/r0raxmg1fg276o05wmmqancki8w-dujour-84c7b497cc-fd7m4
Name:           r0raxmg1fg276o05wmmqancki8w-dujour-84c7b497cc-2trfh
Namespace:      dujour-dev
Node:           gke-shared-k8s-dev-default-pool-e98bef84-n9wk/10.138.0.12
Start Time:     Mon, 07 Jan 2019 13:41:15 -0800
Labels:         app.kubernetes.io/instance=r0raxmg1fg276o05wmmqancki8w
                app.kubernetes.io/name=dujour
                pod-template-hash=4073605377
...
```

Use `top` to monitor resource usage by your compute resources.
```fish
> top
NAME                                                            CPU(cores)  MEMORY(bytes)
/kubernetes/gke_sh...8s-dev/dujour-dev/pods/r0raxm...-fd7m4     3m          296Mi
/aws/resources/ec2/vm-106.puppet.com                            2000m       580Mi
/aws/resources/ec2/vm-107.puppet.com                            190m        540Mi
/aws/resources/lambda/michael-lambda-17                         10m         27Mi
```

### Action

Commands can be run on compute resources via `exec`.

```fish
> exec /kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/pods/r0raxmg1fg276o05wmmqancki8w-dujour-84c7b497cc-fd7m4 /aws/resources/ec2/vm-107.puppet.com -- whoami
Started on r0raxmg1fg276o05wmmqancki8w-dujour-84c7b497cc-fd7m4...
Started on vm-107.puppet.com...
Finished on r0raxmg1fg276o05wmmqancki8w-dujour-84c7b497cc-fd7m4:
  STDOUT:
    guest
Finished on vm-107.puppet.com...
  STDOUT:
    centos
Successful on 2 nodes: r0raxmg1fg276o05wmmqancki8w-dujour-84c7b497cc-fd7m4,vm-107.puppet.com
Ran on 2 nodes in 0.48 seconds
> exec /aws/resources/ec2/vm-106.puppet.com -- do_something >vm-106.json &&
> exec /aws/resources/ec2/vm-107.puppet.com -- do_something_else >vm-107.json &&
> jq vm-106.json vm-107.json
<formatted json>
...
```

We can also edit data such as a ConfigMap, using the program defined in EDITOR.

```fish
> stat /kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/configmap/foo
apiVersion: v1
kind: ConfigMap
metadata:
  creationTimestamp: 2016-02-18T18:52:05Z
  name: game-config
  namespace: default
  resourceVersion: "516"
  selfLink: /api/v1/namespaces/default/configmaps/game-config
  uid: b4952dc3-d670-11e5-8cd0-68f728db1985
> cat /kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/configmap/foo
game.properties: |
  enemies=aliens
  lives=3
  enemies.cheat=true
  enemies.cheat.level=noGoodRotten
  secret.code.passphrase=UUDDLRLRBABAS
  secret.code.allowed=true
  secret.code.lives=30
ui.properties: |
  color.good=purple
  color.bad=yellow
  allow.textmode=true
  how.nice.to.look=fairlyNice
> edit /kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/configmap/foo
```

Or construct a ConfigMap from multiple files.

```fish
> mkdir /kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/configmap/bar
> echo 'enemies=aliens' > /kubernetes/gke_shared-k8s_us-west1-a_shared-k8s-dev/dujour-dev/configmap/bar/game.properties
```

### Troubleshooting

Every action has a history with detailed data. These actions are associated with the objects they act on as well.

So when troubleshooting something that went wrong, we can look at what happened on a particular action in history.

```fish
> inspect 7
Command: ls
Location: /aws/resources/
<http requests to get AWS resources>
> inspect 9
Command: ls
Location: /aws/resources/ec2
<cached data used from AWS>
```

We can also look at the history of our actions on a particular resource.

```fish
> inspect /aws/resources/ec2/vm-107.puppet.com
7: <http request to get AWS resources>
9: <accessed cached resources>
17: <exec debug output from Bolt>
```

## Additional Topics

### Ways of slicing things
- Namespace-oriented? Are namespaces universal? Resource group (AWS), project (GCP), namespace (K8s). Azure has namespaces within resource groups.
- Region. It's not really clear what common semantics exist for this, maybe we should revisit it later.
- Users (or subscription id).
- Cloud API.

### Other questions
- Hardlinks support multiple hierarchical views. What are symlinks?
- What are the types? Compute, storage/volume, database/db. Need a consistent taxonomy, lots of different naming patterns across APIs.
- How should we access details about a particular resource? Metrics?

### Real-world Examples
- Dujour deployment: pod, deployment, chart, pubsub, dataflow, bigquery
- GKE: the k8s infra itself
- GCP: build servers for Pipelines
- Pipelines SaaS: AWS VMs
