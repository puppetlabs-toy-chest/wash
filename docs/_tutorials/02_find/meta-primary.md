---
title: Metadata filtering with the meta primary
---
{% include test_environment_reminder.md %}

This tutorial gives a detailed overview of the `meta` primary. The `meta` primary lets you filter entries on their metadata properties, which is useful when you want to filter entries on a non-Wash attribute property. Examples of such properties include vendor-specific things like an AWS EC2 instance’s VPC ID or a GCP compute instance’s service accounts. They also include properties that don't yet exist as Wash attributes, such as a VM or container’s state, labels, and tags.

The `meta` primary is its own mini-DSL (domain specific language). Each `meta` primary expression consists of:
* The specific metadata property that you’re filtering on
* A predicate on that property’s value. 

There are seven possible value types (excluding `Primitive`):
* `Object`
* `Array`
* `Primitive`
    * `Null`
    * `Boolean`
    * `Numeric`
    * `Time`
    * `String`

Each value type comes with its own predicate. For example, you would construct an `Object Predicate` on `Object` values, an `Array Predicate` on `Array` values, a `Numeric Predicate` on `Numeric` values, or a `Time Predicate` on `Time` values. All predicates evaluate to `false` for mistyped values. For example, an `Object Predicate` would return `false` for a `Numeric` value.

Each value predicate can be combined with another value predicate using `find`’s expression operators. For example, the predicate `+1` is a numeric predicate that means `> 1`. The predicate `+1 -a -3` means `> 1 AND < 3`. Similarly, the predicate `+1 -o -3` means `> 1 OR < 3`.

To facilitate the following discussion, we will pretend that we are filtering on Docker containers. We will use the `wash_tutorial_redis_1` container's metadata when constructing our `meta` primary expressions.

```
wash . ❯ meta docker/containers/wash_tutorial_redis_1
AppArmorProfile: ""
Args:
- redis-server
- --appendonly
- "yes"
Config:
  AttachStderr: false
  AttachStdin: false
  AttachStdout: false
  Cmd:
  - redis-server
  - --appendonly
  - "yes"
  Domainname: ""
  Entrypoint:
  - docker-entrypoint.sh
  Env:
  - PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
  - GOSU_VERSION=1.11
  - REDIS_VERSION=5.0.6
  - REDIS_DOWNLOAD_URL=http://download.redis.io/releases/redis-5.0.6.tar.gz
  - REDIS_DOWNLOAD_SHA=6624841267e142c5d5d5be292d705f8fb6070677687c5aad1645421a936d22b3
  ExposedPorts:
    6379/tcp: {}
  Hostname: 3b197bd973c6
  Image: redis:buster
  Labels:
    com.docker.compose.config-hash: 8c4cc3f3d32489df4e753d5e5fba27ad5f5c139b3858a4df71ef50c0b09b9238
    com.docker.compose.container-number: "1"
    com.docker.compose.oneoff: "False"
    com.docker.compose.project: wash_tutorial
    com.docker.compose.service: redis
    com.docker.compose.version: 1.24.1
  OnBuild: null
  OpenStdin: false
  StdinOnce: false
  Tty: false
  User: ""
  Volumes:
    /data: {}
  WorkingDir: /data
Created: "2019-10-05T18:21:22.7854646Z"
Driver: overlay2
ExecIDs: null
GraphDriver:
  Data:
    LowerDir: /var/lib/docker/overlay2/3982b2fb7f2cfdb74a8ca2344aeee684e1004cd4808c6205de6ff226715e49a9-init/diff:/var/lib/docker/overlay2/8af9dbb0afa2b0eac0b19360f3648d81821905ec2811215d5b4c1a15b5d46585/diff:/var/lib/docker/overlay2/1720b61f5dc9d53bf70613505efe82f7c96a27643688f780d1ded3c9e5ec7662/diff:/var/lib/docker/overlay2/bb1c27bd09c6a39423c065d957733f134f20bfafcbd8d136f796b3c2530b5f49/diff:/var/lib/docker/overlay2/3a1f448d51cbdbc4e6f4b8ff4f7e2b924dd01b382baf72e5987761bc1b570004/diff:/var/lib/docker/overlay2/777dacd8e37b2aa423f4bf565b60b06497d877c2a3cf182746dfae3387341255/diff:/var/lib/docker/overlay2/2b11fabdfd83518a5d386c38da90033e4035ab2969c8fa931f2ecc50d15c05c4/diff
    MergedDir: /var/lib/docker/overlay2/3982b2fb7f2cfdb74a8ca2344aeee684e1004cd4808c6205de6ff226715e49a9/merged
    UpperDir: /var/lib/docker/overlay2/3982b2fb7f2cfdb74a8ca2344aeee684e1004cd4808c6205de6ff226715e49a9/diff
    WorkDir: /var/lib/docker/overlay2/3982b2fb7f2cfdb74a8ca2344aeee684e1004cd4808c6205de6ff226715e49a9/work
  Name: overlay2
HostConfig:
  AutoRemove: false
  Binds:
  - wash_tutorial_redis:/data:rw
  BlkioDeviceReadBps: null
  BlkioDeviceReadIOps: null
  BlkioDeviceWriteBps: null
  BlkioDeviceWriteIOps: null
  BlkioWeight: 0
  BlkioWeightDevice: null
  CapAdd: null
  CapDrop: null
  Capabilities: null
  Cgroup: ""
  CgroupParent: ""
  ConsoleSize:
  - 0
  - 0
  ContainerIDFile: ""
  CpuCount: 0
  CpuPercent: 0
  CpuPeriod: 0
  CpuQuota: 0
  CpuRealtimePeriod: 0
  CpuRealtimeRuntime: 0
  CpuShares: 0
  CpusetCpus: ""
  CpusetMems: ""
  DeviceCgroupRules: null
  DeviceRequests: null
  Devices: null
  Dns: null
  DnsOptions: null
  DnsSearch: null
  ExtraHosts: null
  GroupAdd: null
  IOMaximumBandwidth: 0
  IOMaximumIOps: 0
  IpcMode: shareable
  Isolation: ""
  KernelMemory: 0
  KernelMemoryTCP: 0
  Links: null
  LogConfig:
    Config: {}
    Type: json-file
  MaskedPaths:
  - /proc/asound
  - /proc/acpi
  - /proc/kcore
  - /proc/keys
  - /proc/latency_stats
  - /proc/timer_list
  - /proc/timer_stats
  - /proc/sched_debug
  - /proc/scsi
  - /sys/firmware
  Memory: 0
  MemoryReservation: 0
  MemorySwap: 0
  MemorySwappiness: null
  NanoCpus: 0
  NetworkMode: wash_tutorial_default
  OomKillDisable: false
  OomScoreAdj: 0
  PidMode: ""
  PidsLimit: null
  PortBindings:
    6379/tcp:
    - HostIp: ""
      HostPort: "6379"
  Privileged: false
  PublishAllPorts: false
  ReadonlyPaths:
  - /proc/bus
  - /proc/fs
  - /proc/irq
  - /proc/sys
  - /proc/sysrq-trigger
  ReadonlyRootfs: false
  RestartPolicy:
    MaximumRetryCount: 0
    Name: ""
  Runtime: runc
  SecurityOpt: null
  ShmSize: 67108864
  UTSMode: ""
  Ulimits: null
  UsernsMode: ""
  VolumeDriver: ""
  VolumesFrom: []
HostnamePath: /var/lib/docker/containers/3b197bd973c6f4727f6b029ac52bf3332d964613de07913937583042ea06f153/hostname
HostsPath: /var/lib/docker/containers/3b197bd973c6f4727f6b029ac52bf3332d964613de07913937583042ea06f153/hosts
Id: 3b197bd973c6f4727f6b029ac52bf3332d964613de07913937583042ea06f153
Image: sha256:01a52b3b5cd14dffaff0908e242d11275a682cc8fe3906a0a7ec6f36fbe001f5
LogPath: /var/lib/docker/containers/3b197bd973c6f4727f6b029ac52bf3332d964613de07913937583042ea06f153/3b197bd973c6f4727f6b029ac52bf3332d964613de07913937583042ea06f153-json.log
MountLabel: ""
Mounts:
- Destination: /data
  Driver: local
  Mode: rw
  Name: wash_tutorial_redis
  Propagation: ""
  RW: true
  Source: /var/lib/docker/volumes/wash_tutorial_redis/_data
  Type: volume
Name: /wash_tutorial_redis_1
NetworkSettings:
  Bridge: ""
  EndpointID: ""
  Gateway: ""
  GlobalIPv6Address: ""
  GlobalIPv6PrefixLen: 0
  HairpinMode: false
  IPAddress: ""
  IPPrefixLen: 0
  IPv6Gateway: ""
  LinkLocalIPv6Address: ""
  LinkLocalIPv6PrefixLen: 0
  MacAddress: ""
  Networks:
    wash_tutorial_default:
      Aliases:
      - redis
      - 3b197bd973c6
      DriverOpts: null
      EndpointID: 3237602c94fb0f9838acb0cde2ff19e977c71bab16eec8ba18c1ddefb8405ad1
      Gateway: 172.25.0.1
      GlobalIPv6Address: ""
      GlobalIPv6PrefixLen: 0
      IPAMConfig: null
      IPAddress: 172.25.0.3
      IPPrefixLen: 16
      IPv6Gateway: ""
      Links: null
      MacAddress: 02:42:ac:19:00:03
      NetworkID: f3d6e842b399a3edbd3eab9f37fdf767dd8071e4fff87cea93046d4d3d1c4712
  Ports:
    6379/tcp:
    - HostIp: 0.0.0.0
      HostPort: "6379"
  SandboxID: 39a02f1301effce20d2e478bece77b6be47d8c27c9b69d4e430986c89317d4af
  SandboxKey: /var/run/docker/netns/39a02f1301ef
  SecondaryIPAddresses: null
  SecondaryIPv6Addresses: null
Path: docker-entrypoint.sh
Platform: linux
ProcessLabel: ""
ResolvConfPath: /var/lib/docker/containers/3b197bd973c6f4727f6b029ac52bf3332d964613de07913937583042ea06f153/resolv.conf
RestartCount: 0
SizeRootFs: 98193890
SizeRw: 0
State:
  Dead: false
  Error: ""
  ExitCode: 0
  FinishedAt: "0001-01-01T00:00:00Z"
  OOMKilled: false
  Paused: false
  Pid: 24408
  Restarting: false
  Running: true
  StartedAt: "2019-10-05T18:21:23.8614197Z"
  Status: running
```

**Note:** We’re using the full metadata because a Docker container's `meta` attribute doesn't have enough interesting properties for a meaningful tutorial. In general, you'd _always_ want to use the `meta` attribute’s value instead of the full metadata because filtering on the former is much faster: `O(1)` vs. `O(N)`, where `N` is the number of visited entries.

Now say we wanted to filter on a Docker container's platform. From the `meta` output, we see that `Platform` is the desired property, and the value of that property is a `String`. The latter means that we will be using a `String Predicate`. Thus, the `meta` primary expression would look something like

    -meta '.platform' linux

Let’s try it out!

```
wash . ❯ find docker -fullmeta -k '*container' -meta '.platform' linux
docker/containers/wash_tutorial_redis_1
docker/containers/wash_tutorial_web_1
```

**Note:** The `fullmeta` option tells `find` that we are filtering on the entry's full metadata.

Nice! Note that we used the `kind` primary to explicitly indicate that we are filtering on Docker containers. Also note that the `meta` primary will case property names for you, so you don’t have to be too strict when you’re typing them out. This means that something like `-meta '.PLATFORM' linux` or `-meta '.pLaTfOrM' linux` also work.

Now say we wanted to filter on all of our Docker containers whose maximum retry count is less than 1. From the `meta` output, we see that `hostConfig.restartPolicy.maximumRetryCount` is the desired property, and that this property is a numeric value. Thus, the expression would look something like

    -meta '.hostConfig.restartPolicy.maximumRetryCount' -1

where `-1` means `< 1` (see the section on `Numeric Predicates` in the `meta` primary docs).

Let’s try it out!

```
wash . ❯ find docker -fullmeta -k '*container' -meta '.hostConfig.restartPolicy.maximumRetryCount' -1
docker/containers/wash_tutorial_redis_1
docker/containers/wash_tutorial_web_1
```

Nice!

Now say we wanted to filter on all of our Docker containers that have a mount named `wash_tutorial_redis`. From the `meta` output, we see that this information is contained in the `mounts` property. That property’s value is an array of mounts, and each mount is an object. For a given mount, it looks like the `name` would contain the mount's name. From this information, we see that *Return true if the Docker container has a mount named `wash_tutorial_redis`* is equivalent to *Return true if the Docker container's `mounts` property contains at least one element whose `name` property is `wash_tutorial_redis`*.

Now that we’ve more precisely defined our predicate, it is time to construct the `meta` primary expression. Since `mounts` is our property, we’ll start with `-meta '.mounts'`. Since `mounts` has an `Array` value, that means we must use an `Array Predicate`. Since our `Array Predicate` must return `true` if at least one element matches the final predicate, we’ll use `[?]`. Thus, our expression now becomes `-meta '.mounts' '[?]'`.

Each element in the array is an `Object`. That means we must give our `Array Predicate` an `Object Predicate`. Since the `name` property contains the desired information, our expression becomes `-meta 'mounts' '[?]' '.name'`. Since the `name` property is a `String` value, we must give our `Object Predicate` a `String Predicate`. We want to match on `wash_tutorial_redis`, so our expression now turns into

    -meta '.mounts' '[?]' '.name' wash_tutorial_redis

and we are done. We can shorten this expression to

    -meta '.mounts[?]' '.name' wash_tutorial_redis

or even further to

    -meta '.mounts[?].name' wash_tutorial_redis

So the `find` invocation is something like:

    find docker -fullmeta -k '*container' -meta '.mounts[?].name' wash_tutorial_redis

Try it out and see if it works.

That wraps up our discussion on the meta primary’s DSL. We’ll conclude this tutorial by giving you a general overview of how to construct a meta primary expression. Assuming you have a general idea of the specific property that you want to filter on, then here’s what you should do:

1. Find a representative entry that you can use to construct your expression. In our examples, we chose the `wash_tutorial_redis` container.

2. Check that entry’s `meta` attribute value via `meta --attribute` and see if the desired property is there. If it is, then figure out the property value’s type and construct the appropriate predicate. Remember that you can use `find --help meta` to view the `meta` primary’s full documentation.

3. If the `meta` attribute does not contain the desired property, then you’ll have to check the entry’s full metadata via `meta`. If the full metadata contains the property, then refer to Step 2 and be sure to pass the `-fullmeta` option to `find`’s invocation so that `find` knows that it’ll need to fetch the entry’s full metadata. If the full metadata does not contain the property, then you’ll have to contact the plugin author(s) and ask them if they could include your property’s value in the entry’s metadata.

# Exercises

{% include exercise_reminder.md %}

1. This exercise is broken up into several parts. Each part will ask you to filter on something specific to a Docker container. Your job is to provide the appropriate `meta` primary expression that accomplishes the given task. For example, a valid answer for "Has `N < 1` maximum retry counts" would be `-meta '.hostConfig.restartPolicy.maximumRetryCount' -1`. Hint: Use the example’s metadata. Remember to set the `fullmeta` option when testing your expressions.

    1. Has `N == 0 AND N < 1` maximum retry counts.

        {% include exercise_answer.html answer="<code>-meta '.hostConfig.restartPolicy.maximumRetryCount' 0 -a -1</code>" %}

    2. Has a mount whose name is either `wash_tutorial_redis` or `wash_tutorial_web`

        {% include exercise_answer.html answer="<code>-meta '.mounts[?]' '.name' wash_tutorial_redis -o wash_tutorial_web</code>" %}

    3. Was started within the last 4 weeks.

        {% include exercise_answer.html answer="<code>-meta '.state.startedAt' -4w</code>" %}

    7. Is not in the `stopped` state.

        {% include exercise_answer.html answer="<code>-meta '.state.status' \! stopped</code>" %}

    8. Has the `wash_tutorial_redis` _volume_ mounted. Hint: The predicate is still constructed on the `mounts` property. What’s the property that corresponds to a mount's type?

        {% include exercise_answer.html answer="<code>-meta '.mounts[?]' '.name' wash_tutorial_redis -a '.type' volume</code>" %}

2. This exercise is broken up into several parts. Each part will ask you to find entries that satisfy a specific set of criteria. Your job is to provide the appropriate `find` invocation that accomplishes the given task. For example, a valid answer for "Find all EC2 instances with the `project` tag in a given profile" would be `find aws/<profile> -k '*ec2*instance' -meta '.tags[?]' '.key' project`. Hint: The `meta` attribute should contain what you need.

   **Note:** Even if you're not using a given plugin, we recommend that you take a look at these answers to see the full extent of the `meta` primary's power.

    1. Find all running Docker containers.

        {% include exercise_answer.html answer="<code>find docker -k '*container' -meta '.state' running</code>" %}

    2. Find all running GCP compute instances in a given project.

        {% include exercise_answer.html answer="<code>find gcp/&lt;project&gt; -k '*compute*instance' -meta '.status' RUNNING</code>" %}

    3. Find all running Kubernetes pods in a given context.

        {% include exercise_answer.html answer="<code>find kubernetes/&lt;context&gt; -k '*pod' -meta '.status.phase' Running</code>" %}

    4. Find all GCP storage buckets in a given project that are using `REGIONAL` storage.

        {% include exercise_answer.html answer="<code>find gcp/&lt;project&gt; -k '*storage*bucket' -meta '.storageClass' REGIONAL</code>. Remember that the concepts in this tutorial apply to any entry, not just containers/VMs" %}

    5. Find all running Docker containers that were created within the last 24 hours. Hint: Remember the `crtime` primary and `find`’s operators.

        {% include exercise_answer.html answer="<code>find docker -k '*container' -meta '.state' running -crtime -24h</code>" %}

    6. Find all running AWS EC2 instances in a given profile that were launched more than a week ago. Hint: You’ll need to use the `meta` primary more than once here.

        {% include exercise_answer.html answer="<code>find aws/&lt;profile&gt; -k '*ec2*instance' -meta '.state.name' running -meta '.launchTime' +1w</code>" %}

    7. Find all running GCP compute instances in a given project whose `owner` label is set to `jimmy`.

        {% include exercise_answer.html answer="<code>find gcp/&lt;project&gt; -k '*compute*instance' -meta '.status' RUNNING -meta '.labels.owner' jimmy</code>" %}

    8. Find all Kubernetes pods in a given context that have the `owner` label. Hint: It is enough to check that the label exists.

        {% include exercise_answer.html answer="<code>find kubernetes/&lt;context&gt; -k '*pod' -meta '.metadata.labels.owner' -exists</code>" %}

# Next steps

That's the end of the _Filtering entries with find_ series! Click [here](../) to go back to the tutorials page.
