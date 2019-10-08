---
title: Understanding attributes and metadata
---
This tutorial covers attributes and metadata, which are primarily used for filtering entries with the `find` command. The `find` command is introduced in [Filtering entries with find]({{ '/tutorials/02_find' | relative_url }}).

# Metadata
All entries are completely described by their metadata. An entry’s metadata is a key-value map containing everything you’ll ever need to know about the entry. You can use the `meta` command to view an entry’s metadata.

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
  Hostname: b7773fcfb315
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
Created: "2019-10-04T22:30:05.9409287Z"
Driver: overlay2
ExecIDs: null
GraphDriver:
  Data:
    LowerDir: /var/lib/docker/overlay2/e04d8cb6e8cdc2a7ad9cbcaaaeef07372ec154b65c8016716953fa5168797417-init/diff:/var/lib/docker/overlay2/8af9dbb0afa2b0eac0b19360f3648d81821905ec2811215d5b4c1a15b5d46585/diff:/var/lib/docker/overlay2/1720b61f5dc9d53bf70613505efe82f7c96a27643688f780d1ded3c9e5ec7662/diff:/var/lib/docker/overlay2/bb1c27bd09c6a39423c065d957733f134f20bfafcbd8d136f796b3c2530b5f49/diff:/var/lib/docker/overlay2/3a1f448d51cbdbc4e6f4b8ff4f7e2b924dd01b382baf72e5987761bc1b570004/diff:/var/lib/docker/overlay2/777dacd8e37b2aa423f4bf565b60b06497d877c2a3cf182746dfae3387341255/diff:/var/lib/docker/overlay2/2b11fabdfd83518a5d386c38da90033e4035ab2969c8fa931f2ecc50d15c05c4/diff
    MergedDir: /var/lib/docker/overlay2/e04d8cb6e8cdc2a7ad9cbcaaaeef07372ec154b65c8016716953fa5168797417/merged
    UpperDir: /var/lib/docker/overlay2/e04d8cb6e8cdc2a7ad9cbcaaaeef07372ec154b65c8016716953fa5168797417/diff
    WorkDir: /var/lib/docker/overlay2/e04d8cb6e8cdc2a7ad9cbcaaaeef07372ec154b65c8016716953fa5168797417/work
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
HostnamePath: /var/lib/docker/containers/b7773fcfb315c3d230226f2f13aebd309473730342ba23df3a19251147eb98c9/hostname
HostsPath: /var/lib/docker/containers/b7773fcfb315c3d230226f2f13aebd309473730342ba23df3a19251147eb98c9/hosts
Id: b7773fcfb315c3d230226f2f13aebd309473730342ba23df3a19251147eb98c9
Image: sha256:01a52b3b5cd14dffaff0908e242d11275a682cc8fe3906a0a7ec6f36fbe001f5
LogPath: /var/lib/docker/containers/b7773fcfb315c3d230226f2f13aebd309473730342ba23df3a19251147eb98c9/b7773fcfb315c3d230226f2f13aebd309473730342ba23df3a19251147eb98c9-json.log
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
      - b7773fcfb315
      - redis
      DriverOpts: null
      EndpointID: e87c4268fb2aa8e3e5425e28537711d96d11f475726f9ec6f06aa933992c63b5
      Gateway: 172.23.0.1
      GlobalIPv6Address: ""
      GlobalIPv6PrefixLen: 0
      IPAMConfig: null
      IPAddress: 172.23.0.3
      IPPrefixLen: 16
      IPv6Gateway: ""
      Links: null
      MacAddress: 02:42:ac:17:00:03
      NetworkID: 7b2574661bec274e0b23a52b942389734f6d76112bb6fc0017e3446273b38885
  Ports:
    6379/tcp:
    - HostIp: 0.0.0.0
      HostPort: "6379"
  SandboxID: 45b73dd2762f3a1ea3ce575469ed47d1c6a58a7940b143f90b7a2a3959d9f471
  SandboxKey: /var/run/docker/netns/45b73dd2762f
  SecondaryIPAddresses: null
  SecondaryIPv6Addresses: null
Path: docker-entrypoint.sh
Platform: linux
ProcessLabel: ""
ResolvConfPath: /var/lib/docker/containers/b7773fcfb315c3d230226f2f13aebd309473730342ba23df3a19251147eb98c9/resolv.conf
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
  Pid: 18809
  Restarting: false
  Running: true
  StartedAt: "2019-10-04T22:30:06.936981432Z"
  Status: running
wash . ❯
```

As you can see, a Docker container is described by quite a few properties. For example, the bottom part of the output shows us these properties: 
* `Platform`
* `ProcessLabel`
* `ResolvConfPath`
* `RestartCount`
* `SizeRootFs`
* `SizeRw`
* `State`

We see that the values for each of these properties is also included. For example, the value of the `Platform` property is `linux`, which tells us that this container is a Linux container.

## Exercises

{% include exercise_reminder.md %}

1. Using the above meta output, what are the values of the following properties?
    1. `SizeRootFs`
    1. `Created`
    1. `State.StartedAt`
    1. `Args[0]`
    1. `HostConfig.ReadonlyPaths[0]`

    {% capture answer_1 %}
      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;a. <code>98193890</code><br />
      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;b. <code>"2019-10-04T22:30:05.9409287Z"</code><br />
      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;c. <code>"2019-10-04T22:30:06.936981432Z"</code><br />
      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;d. <code>redis-server</code><br />
      &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;e. <code>/proc/bus</code>
    {% endcapture %}
    {% include exercise_answer.html answer=answer_1 %}

1. The `aws/<profile>/resources/ec2/instances` directory contains all the EC2 instances that are accessible by the `<profile>` profile. What property contains an EC2 instance's

    1. Tags?

       {% include exercise_answer.html answer="<code>Tags</code>" %}

    1. Current, human-readable state?

       {% include exercise_answer.html answer="<code>State.Name</code>" %}

    1. VPC ID?

       {% include exercise_answer.html answer="<code>VpcId</code>" %}

    1. Security groups? For a given security group, what property contains its human-readable name?

       {% include exercise_answer.html answer="<code>SecurityGroups</code> contains the instance's security groups. For a given security group, the <code>GroupName</code> property contains its human-readable name." %}

1. The `gcp/<project>/storage/<bucket>` directory contains all the GCP storage objects of the `<bucket>` bucket in the `<project>` project. What property contains a storage object's

    1. Storage class?

       {% include exercise_answer.html answer="<code>StorageClass</code>" %}

    1. Content-type?

       {% include exercise_answer.html answer="<code>ContentType</code>" %}

You might have found Exercise 3 similar to Exercise 2. This was on purpose. The point of Exercise 3 was to reinforce the fact that the metadata concept applies to any entry, including entries as diverse as AWS EC2 instances and GCP Storage objects.

# Attributes
Some metadata properties like creation time (`crtime`), last modified time (`mtime`), and content size (`size`) are common across entries. These common properties make up an entry’s attributes. From its definition, we see that an entry’s attributes are a subset of its metadata.

You can use `winfo` to view an entry’s attributes.

```
wash . ❯ winfo docker/containers/wash_tutorial_redis_1
Path: /Users/enis.inan/Library/Caches/wash/mnt184960766/docker/containers/wash_tutorial_redis_1
Name: wash_tutorial_redis_1
CName: wash_tutorial_redis_1
Actions:
- list
- exec
Attributes:
  atime: 2019-10-04T15:30:05-07:00
  crtime: 2019-10-04T15:30:05-07:00
  ctime: 2019-10-04T15:30:05-07:00
  mtime: 2019-10-04T15:30:05-07:00
```

Here, we see that a Docker container has the `atime`, `crtime`, `ctime` and `mtime` attributes, and that for this particular container, these attributes are all set to `2019-10-04T15:30:05-07:00`.

Every entry also includes a special `meta` attribute. The `meta` attribute is the subset of the entry’s metadata that the plugin’s API returns when you attempt to list that entry’s parent (think of it as the entry’s ‘partial’ metadata). For example, a Docker container’s `meta` attribute is the raw JSON object returned by Docker’s `/containers/json` endpoint. Wash hits that endpoint when you attempt to `ls` the `docker/containers` directory.

You can use the `meta` command’s `--attribute` option to view an entry’s `meta` attribute.

```
wash . ❯ meta --attribute docker/containers/wash_tutorial_redis_1
Command: docker-entrypoint.sh redis-server --appendonly yes
Created: 1570228205
HostConfig:
  NetworkMode: wash_tutorial_default
Id: b7773fcfb315c3d230226f2f13aebd309473730342ba23df3a19251147eb98c9
Image: redis:buster
ImageID: sha256:01a52b3b5cd14dffaff0908e242d11275a682cc8fe3906a0a7ec6f36fbe001f5
Labels:
  com.docker.compose.config-hash: 8c4cc3f3d32489df4e753d5e5fba27ad5f5c139b3858a4df71ef50c0b09b9238
  com.docker.compose.container-number: "1"
  com.docker.compose.oneoff: "False"
  com.docker.compose.project: wash_tutorial
  com.docker.compose.service: redis
  com.docker.compose.version: 1.24.1
Mounts:
- Destination: /data
  Driver: local
  Mode: rw
  Name: wash_tutorial_redis
  Propagation: ""
  RW: true
  Source: /var/lib/docker/volumes/wash_tutorial_redis/_data
  Type: volume
Names:
- /wash_tutorial_redis_1
NetworkSettings:
  Networks:
    wash_tutorial_default:
      Aliases: null
      DriverOpts: null
      EndpointID: e87c4268fb2aa8e3e5425e28537711d96d11f475726f9ec6f06aa933992c63b5
      Gateway: 172.23.0.1
      GlobalIPv6Address: ""
      GlobalIPv6PrefixLen: 0
      IPAMConfig: null
      IPAddress: 172.23.0.3
      IPPrefixLen: 16
      IPv6Gateway: ""
      Links: null
      MacAddress: 02:42:ac:17:00:03
      NetworkID: 7b2574661bec274e0b23a52b942389734f6d76112bb6fc0017e3446273b38885
Ports:
- IP: 0.0.0.0
  PrivatePort: 6379
  PublicPort: 6379
  Type: tcp
State: running
Status: Up 3 hours
```

Comparing this output with the `meta` command’s output in the Metadata section, we see that a Docker container’s `meta` attribute includes far fewer properties than its metadata. That’s because Docker’s `containers/json` endpoint’s response doesn’t include all of a given container’s information.

## Exercises

{% include exercise_reminder.md %}

1. What are the attributes of an AWS EC2 instance?

   {% include exercise_answer.html answer="<code>crtime</code> and <code>mtime</code>" %}

1. What are the attributes of a GCP storage object?

   {% include exercise_answer.html answer="<code>crtime</code>, <code>ctime</code>, <code>mtime</code> and <code>size</code>" %}

## Related Links
* Check out the [attributes docs]({{ '/docs#attributes' | relative_url }}) to see all of the available Wash attributes.

# Next steps

Now that you've learned about attributes and metadata, you can move on to learn about viewing a [plugin's documentation](documentation).
