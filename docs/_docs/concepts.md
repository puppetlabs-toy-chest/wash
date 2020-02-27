---
title: Concepts
---
* [CName](#cname)
  * [Examples](#examples)
* [Actions](#actions)
  * [list](#list)
    * [Examples](#examples-1)
  * [read](#read)
    * [Examples](#examples-2)
  * [write](#write)
    * [Examples](#examples-3)
  * [stream](#stream)
    * [Examples](#examples-4)
  * [exec](#exec)
    * [Examples](#examples-5)
  * [delete](#delete)
    * [Examples](#examples-6)
  * [signal](#signal)
    * [Examples](#examples-7)
    * [Common Signals](#common-signals)
* [Attributes](#attributes)
  * [crtime](#crtime)
    * [Example JSON](#example-json)
  * [mtime](#mtime)
    * [Example JSON](#example-json-1)
  * [ctime](#ctime)
    * [Example JSON](#example-json-2)
  * [atime](#atime)
    * [Example JSON](#example-json-3)
  * [size](#size)
    * [Example JSON](#example-json-4)
  * [mode](#mode)
    * [Example JSON](#example-json-5)
  * [os](#os)
    * [Example JSON](#example-json-6)

## CName

CName is short for _canonical name_. An entry's cname is its name with all `/`'es replaced by that entry's _slash replacer_. The default slash replacer is `#`.

Wash uses the cname to construct the entry's path. An entry's path is defined as
```
    <mountpoint>/<parent_cname1>/<parent_cname2>/.../<cname>
```

### Examples
Consider the entry `/myplugin/foo`. If `bar/baz` is a child of `foo`, then its cname and path would be `bar#baz` and `/myplugin/foo/bar#baz`, respectively. Similarly if `qux` is a child of `foo`, then its cname and path would be `qux` and `/myplugin/foo/qux`.

Conversely, if `bar/baz`'s slash replacer is set to `:`, then its cname and path would now be `bar:baz` and `/myplugin/foo/bar:baz`.

## Actions

### list
The `list` action lets you list an entry’s children. Entries that support list are represented as directories. Thus, any command that works with directories also work with these entries.

#### Examples
```
wash . ❯ ls gcp/Wash/storage/some-wash-stuff
an example folder reaper.sh
```

```
wash . ❯ cd gcp/Wash/storage/some-wash-stuff
wash gcp/Wash/storage/some-wash-stuff ❯
```

```
wash . ❯ tree gcp/Wash/storage/some-wash-stuff
gcp/Wash/storage/some-wash-stuff
├── an\ example\ folder
│   └── static.sh
└── reaper.sh

1 directory, 2 files
```

### read
The `read` action lets you read data from an entry. Thus, any command that reads a file also works with these entries.

#### Examples
```
wash . ❯ cat gcp/Wash/storage/some-wash-stuff/an\ example\ folder/static.sh
#!/bin/sh

echo "Hello, world!"
```

```
wash . ❯ grep "Hello" gcp/Wash/storage/some-wash-stuff/an\ example\ folder/static.sh
echo "Hello, world!"
```

### write
The `write` action lets you write data to an entry. Thus, any command that writes a file also works with these entries.

Note that Wash distinguishes between file-like and non-file-like entries. An entry is file-like if it's readable and writable and defines its size; you can edit it like a file.

If it doesn't define a size then it's non-file-like, and trying to open it with a ReadWrite handle will error; reads from it may not return data you previously wrote to it. You should check its documentation with the `docs` command for that entry's write semantics. We also recommend not using editors with these entries to avoid weird behavior.

#### Examples
Modifying a file stored in Google Cloud Storage
```
wash . ❯ echo 'exit 1' >> gcp/Wash/storage/some-wash-stuff/an\ example\ folder/static.sh
wash . ❯ cat gcp/Wash/storage/some-wash-stuff/an\ example\ folder/static.sh
#!/bin/sh

echo "Hello, world!"
exit 1
```

Writing a message to a hypothetical message queue where each write publishes a message and each read consumes a message
```
wash > echo 'message 1' >> myqueue
wash > echo 'message 2' >> myqueue
wash > cat myqueue
message 1
wash > cat myqueue
message 2
```

### stream
The `stream` action lets you stream an entry’s content for updates.

#### Examples
```
wash . ❯ tail -f gcp/Wash/compute/instance-1/fs/var/log/messages
===> gcp/Wash/compute/instance-1/fs/var/log/messages <===
Aug 27 06:25:01 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="17499" x-info="http://www.rsyslog.com"] rsyslogd was HUPed

Aug 27 13:26:32 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="17499" x-info="http://www.rsyslog.com"] exiting on signal 15.

Aug 27 13:26:32 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="24583" x-info="http://www.rsyslog.com"] start
Aug 28 00:30:04 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="24583" x-info="http://www.rsyslog.com"] exiting on signal 15.
Aug 28 00:30:04 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="32147" x-info="http://www.rsyslog.com"] start
Aug 28 06:25:01 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="32147" x-info="http://www.rsyslog.com"] rsyslogd was HUPed

Aug 28 09:54:34 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="32147" x-info="http://www.rsyslog.com"] exiting on signal 15.

Aug 28 09:54:34 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="6687" x-info="http://www.rsyslog.com"] start
Aug 28 19:01:21 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="6687" x-info="http://www.rsyslog.com"] exiting on signal 15.

Aug 28 19:01:21 instance-1 liblogging-stdlog:  [origin software="rsyslogd" swVersion="8.24.0" x-pid="12804" x-info="http://www.rsyslog.com"] start
```

(Hit `Ctrl+C` to cancel `tail -f`)

### exec
The `exec` action lets you execute a command on an entry.

#### Examples
```
wash . ❯ wexec gcp/Wash/compute/instance-1 uname
Linux
```

### delete
The `delete` action lets you delete an entry.

#### Examples
```
wash . ❯ delete docker/containers/quizzical_colden
remove docker/containers/quizzical_colden?: y
```

### signal
The `signal` action lets you signal an entry. Use the `docs` command to view an entry's supported signals.

#### Examples
```
wash . ❯ docs docker/containers/wash_tutorial_redis_1
No description provided.

SUPPORTED SIGNALS
* start
    Starts the container. Equivalent to 'docker start <container>'
* stop
    Stops the container. Equivalent to 'docker stop <container>'
* pause
    Suspends all processes in the container. Equivalent to 'docker pause <container>'
* resume
    Un-suspends all processes in the container. Equivalent to 'docker unpause <container>'
* restart
    Restarts the container. Equivalent to 'docker restart <container>'

SUPPORTED SIGNAL GROUPS
* linux
    Consists of all the supported Linux signals like SIGHUP, SIGKILL. Equivalent to
    'docker kill <container> --signal <signal>'
```

```
wash . ❯ signal start docker/containers/wash_tutorial_redis_1
wash . ❯
```

#### Common Signals
* start
* stop
* pause
* resume
* restart
* hibernate
* reset

## Attributes

### crtime
This is the entry's creation time.

#### Example JSON
As a stringified date

```
{
  "crtime": “2019-09-25T21:39:57-07:00”
}
```

In UNIX seconds

```
{
  "crtime": 1569472797
}
```

### mtime
This is the entry's last modification time.

#### Example JSON
As a stringified date

```
{
  "mtime": “2019-09-25T21:39:57-07:00”
}
```

In UNIX seconds

```
{
  "mtime": 1569472797
}
```

### ctime
This is the entry's last change time.

#### Example JSON
As a stringified date

```
{
  "ctime": “2019-09-25T21:39:57-07:00”
}
```

In UNIX seconds

```
{
  "ctime": 1569472797
}
```

### atime
This is the entry's last access time.

#### Example JSON
As a stringified date

```
{
  "atime": “2019-09-25T21:39:57-07:00”
}
```

In UNIX seconds

```
{
  "atime": 1569472797
}
```

### size
This is the entry's content size.

#### Example JSON
```
{
  "size": 1024
}
```

### mode
This is the entry's mode.

#### Example JSON

```
{
  "mode": 16832
}
```

As a hexadecimal string

```
{
  "mode": "41C0"
}
```

As an octal string

```
{
  "mode": "40700"
}
```

### os
This contains information about the operating system of an entry, if it has one.

The `OS` property is an object containing
* `login_shell` - The shell used when remotely logging into the machine, such as over SSH or WinRM. It can be one of `posixshell` (representing a POSIX-compatible shell such as bash or zsh) or `powershell` (PowerShell). This is commonly used to decide what types of shell is available for scripting; for example the `wps` command runs different commands to find process information based on the login shell.

#### Example JSON

```
{
  "os": {
    "login_shell": "posixshell"
  }
}
```
