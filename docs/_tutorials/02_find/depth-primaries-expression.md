---
title: Understanding depth, primaries and expression syntax
---
**Note:** If you’ve used BSD/GNU’s `find` command, then much of the stuff in this tutorial will be familiar to you.

The `find` command recursively descends a given path, printing out all of its subchildren.

```
wash . ❯ find docker
docker
docker/containers
docker/containers/wash_tutorial_redis_1
docker/containers/wash_tutorial_redis_1/fs
docker/containers/wash_tutorial_redis_1/fs/.dockerenv
docker/containers/wash_tutorial_redis_1/fs/bin
docker/containers/wash_tutorial_redis_1/fs/bin/bash
docker/containers/wash_tutorial_redis_1/fs/bin/cat
docker/containers/wash_tutorial_redis_1/fs/bin/chgrp
docker/containers/wash_tutorial_redis_1/fs/bin/chmod
docker/containers/wash_tutorial_redis_1/fs/bin/chown
docker/containers/wash_tutorial_redis_1/fs/bin/cp
docker/containers/wash_tutorial_redis_1/fs/bin/dash
docker/containers/wash_tutorial_redis_1/fs/bin/date
docker/containers/wash_tutorial_redis_1/fs/bin/dd
docker/containers/wash_tutorial_redis_1/fs/bin/df
docker/containers/wash_tutorial_redis_1/fs/bin/dir
docker/containers/wash_tutorial_redis_1/fs/bin/dmesg
docker/containers/wash_tutorial_redis_1/fs/bin/dnsdomainname
docker/containers/wash_tutorial_redis_1/fs/bin/domainname
docker/containers/wash_tutorial_redis_1/fs/bin/echo
docker/containers/wash_tutorial_redis_1/fs/bin/egrep
docker/containers/wash_tutorial_redis_1/fs/bin/false
docker/containers/wash_tutorial_redis_1/fs/bin/fgrep
docker/containers/wash_tutorial_redis_1/fs/bin/findmnt
docker/containers/wash_tutorial_redis_1/fs/bin/grep
docker/containers/wash_tutorial_redis_1/fs/bin/gunzip
docker/containers/wash_tutorial_redis_1/fs/bin/gzexe
docker/containers/wash_tutorial_redis_1/fs/bin/gzip
docker/containers/wash_tutorial_redis_1/fs/bin/hostname
docker/containers/wash_tutorial_redis_1/fs/bin/ln
docker/containers/wash_tutorial_redis_1/fs/bin/login
docker/containers/wash_tutorial_redis_1/fs/bin/ls
docker/containers/wash_tutorial_redis_1/fs/bin/lsblk
docker/containers/wash_tutorial_redis_1/fs/bin/mkdir
docker/containers/wash_tutorial_redis_1/fs/bin/mknod
docker/containers/wash_tutorial_redis_1/fs/bin/mktemp
docker/containers/wash_tutorial_redis_1/fs/bin/more
docker/containers/wash_tutorial_redis_1/fs/bin/mount
docker/containers/wash_tutorial_redis_1/fs/bin/mountpoint
docker/containers/wash_tutorial_redis_1/fs/bin/mv
docker/containers/wash_tutorial_redis_1/fs/bin/nisdomainname
docker/containers/wash_tutorial_redis_1/fs/bin/pidof
docker/containers/wash_tutorial_redis_1/fs/bin/pwd
docker/containers/wash_tutorial_redis_1/fs/bin/rbash
docker/containers/wash_tutorial_redis_1/fs/bin/readlink
docker/containers/wash_tutorial_redis_1/fs/bin/rm
docker/containers/wash_tutorial_redis_1/fs/bin/rmdir
docker/containers/wash_tutorial_redis_1/fs/bin/run-parts
docker/containers/wash_tutorial_redis_1/fs/bin/sed
docker/containers/wash_tutorial_redis_1/fs/bin/sh
docker/containers/wash_tutorial_redis_1/fs/bin/sleep
docker/containers/wash_tutorial_redis_1/fs/bin/stty
docker/containers/wash_tutorial_redis_1/fs/bin/su
docker/containers/wash_tutorial_redis_1/fs/bin/sync
docker/containers/wash_tutorial_redis_1/fs/bin/tar
docker/containers/wash_tutorial_redis_1/fs/bin/tempfile
docker/containers/wash_tutorial_redis_1/fs/bin/touch
docker/containers/wash_tutorial_redis_1/fs/bin/true
docker/containers/wash_tutorial_redis_1/fs/bin/umount
docker/containers/wash_tutorial_redis_1/fs/bin/uname
docker/containers/wash_tutorial_redis_1/fs/bin/uncompress
docker/containers/wash_tutorial_redis_1/fs/bin/vdir
docker/containers/wash_tutorial_redis_1/fs/bin/wdctl
docker/containers/wash_tutorial_redis_1/fs/bin/which
docker/containers/wash_tutorial_redis_1/fs/bin/ypdomainname
docker/containers/wash_tutorial_redis_1/fs/bin/zcat
docker/containers/wash_tutorial_redis_1/fs/bin/zcmp
docker/containers/wash_tutorial_redis_1/fs/bin/zdiff
docker/containers/wash_tutorial_redis_1/fs/bin/zegrep
docker/containers/wash_tutorial_redis_1/fs/bin/zfgrep
docker/containers/wash_tutorial_redis_1/fs/bin/zforce
docker/containers/wash_tutorial_redis_1/fs/bin/zgrep
docker/containers/wash_tutorial_redis_1/fs/bin/zless
docker/containers/wash_tutorial_redis_1/fs/bin/zmore
docker/containers/wash_tutorial_redis_1/fs/bin/znew
docker/containers/wash_tutorial_redis_1/fs/boot
docker/containers/wash_tutorial_redis_1/fs/data
docker/containers/wash_tutorial_redis_1/fs/data/appendonly.aof
docker/containers/wash_tutorial_redis_1/fs/dev
docker/containers/wash_tutorial_redis_1/fs/dev/core
docker/containers/wash_tutorial_redis_1/fs/dev/fd
docker/containers/wash_tutorial_redis_1/fs/dev/fd/0
docker/containers/wash_tutorial_redis_1/fs/dev/fd/1
docker/containers/wash_tutorial_redis_1/fs/dev/fd/2
docker/containers/wash_tutorial_redis_1/fs/dev/full
docker/containers/wash_tutorial_redis_1/fs/dev/mqueue
docker/containers/wash_tutorial_redis_1/fs/dev/null
docker/containers/wash_tutorial_redis_1/fs/dev/ptmx
docker/containers/wash_tutorial_redis_1/fs/dev/pts
docker/containers/wash_tutorial_redis_1/fs/dev/pts/0
docker/containers/wash_tutorial_redis_1/fs/dev/pts/ptmx
docker/containers/wash_tutorial_redis_1/fs/dev/random
docker/containers/wash_tutorial_redis_1/fs/dev/shm
docker/containers/wash_tutorial_redis_1/fs/dev/stderr
docker/containers/wash_tutorial_redis_1/fs/dev/stdin
docker/containers/wash_tutorial_redis_1/fs/dev/stdout
docker/containers/wash_tutorial_redis_1/fs/dev/tty
docker/containers/wash_tutorial_redis_1/fs/dev/urandom
docker/containers/wash_tutorial_redis_1/fs/dev/zero
docker/containers/wash_tutorial_redis_1/fs/etc
docker/containers/wash_tutorial_redis_1/fs/etc/.pwd.lock
docker/containers/wash_tutorial_redis_1/fs/etc/adduser.conf
docker/containers/wash_tutorial_redis_1/fs/etc/alternatives
docker/containers/wash_tutorial_redis_1/fs/etc/alternatives/README
docker/containers/wash_tutorial_redis_1/fs/etc/alternatives/awk
docker/containers/wash_tutorial_redis_1/fs/etc/alternatives/nawk
docker/containers/wash_tutorial_redis_1/fs/etc/alternatives/pager
docker/containers/wash_tutorial_redis_1/fs/etc/alternatives/rmt
docker/containers/wash_tutorial_redis_1/fs/etc/apt
docker/containers/wash_tutorial_redis_1/fs/etc/apt/apt.conf.d
docker/containers/wash_tutorial_redis_1/fs/etc/apt/apt.conf.d/01autoremove
docker/containers/wash_tutorial_redis_1/fs/etc/apt/apt.conf.d/70debconf
docker/containers/wash_tutorial_redis_1/fs/etc/apt/apt.conf.d/docker-autoremove-suggests
docker/containers/wash_tutorial_redis_1/fs/etc/apt/apt.conf.d/docker-clean
docker/containers/wash_tutorial_redis_1/fs/etc/apt/apt.conf.d/docker-gzip-indexes
docker/containers/wash_tutorial_redis_1/fs/etc/apt/apt.conf.d/docker-no-languages
docker/containers/wash_tutorial_redis_1/fs/etc/apt/auth.conf.d
docker/containers/wash_tutorial_redis_1/fs/etc/apt/preferences.d
^C
```

If we let `find` run indefinitely, the output would contain every entry in the Docker plugin. We prematurely stopped it here because `find` was descending into the `wash_tutorial_redis_1` container's root directory, and enumerating root directories takes a long time.

We can use the `maxdepth` option to limit `find`’s recursion.

```
wash . ❯ find docker -maxdepth 1
docker
docker/containers
docker/volumes
```

Note that the depth starts from `0` and is relative to the specified path. Thus, the `docker` entry has depth `0`. The `docker/containers` and `docker/volumes` entries both have depth `1`.

```
wash . ❯ find docker -maxdepth 2
docker
docker/containers
docker/containers/wash_tutorial_redis_1
docker/containers/wash_tutorial_web_1
docker/volumes
docker/volumes/wash_tutorial_redis
```

Note that the `docker`, `docker/containers`, `docker/volumes` entries still have depth `0`, `1`, `1`, respectively. However, the `docker/containers/<container>` and `docker/volumes/<volume>` entries have depth `2`, which is also within the `maxdepth` of `2`. Thus, the above invocation printed out all Docker containers and volumes.

The `find` command also takes multiple paths.

```
wash . ❯ find docker/containers docker/volumes -maxdepth 1
docker/containers
docker/containers/wash_tutorial_redis_1
docker/containers/wash_tutorial_web_1
docker/volumes
docker/volumes/wash_tutorial_redis
```

In many Unix-like operating systems, the `find` command is primarily used to filter files and directories that satisfy a predicate, where a predicate is a statement that returns `true` or `false` for a given file/directory. For example, `find /var/log -name '*.log' -mtime -1h` would display all `.log` files in `/var/log` that were modified within the last hour.

Similarly, in Wash, the `find` command is primarily used to filter entries that satisfy a predicate, where a predicate is a statement that returns `true` or `false` for a given entry. For example, `find docker/containers/wash_tutorial_redis_1/fs/var/log -name '*.log' -mtime -1h` will display all `.log` files in the `wash_tutorial_redis_1` container’s `/var/log` directory that were modified in the last hour.

`find` supports an expression syntax that lets you construct your predicate using primaries (the individual predicates) and operators (the things that combine the predicates together). For example, the `name` primary takes in a glob and returns true if the entry’s `cname`[^1] matches that glob (and false otherwise).

[^1]: See the [cname docs]({{ '/docs#cname' | relative_url }}) to learn more about entry cnames.

```
wash . ❯ find docker/containers -maxdepth 1 -name 'wash_tutorial*'
docker/containers/wash_tutorial_redis_1
docker/containers/wash_tutorial_web_1
```

This printed out all containers whose cname matched the `wash_tutorial*` glob (all containers that start with `wash_tutorial`). Note that the `maxdepth` option is there to prevent recursing into a container. The recursion happens because every entry has a `cname`, so it is possible for some other entry’s `cname` to match the provided glob. For example, a file or directory that’s inside the container.

Similarly, the `-o` operator takes two predicates `p1` and `p2` and returns `p1 OR p2`, where `OR` is the logical `OR` operator found in programming languages like C++ or Java.

```
wash . ❯ find docker/containers -maxdepth 1 -name '*web*' -o -name '*redis*'
docker/containers/wash_tutorial_redis_1
docker/containers/wash_tutorial_web_1
```

This printed out all containers whose cname matched the `*web*` glob OR whose cname matches the `*redis*` glob. In other words, the command finds all containers that manage `web` and `redis`.

You can use `find --help` to view all the available primaries and operators; `find --help <primary>` to get a more detailed overview of a given primary; and `find --help syntax` to get a more detailed overview of `find`’s expression syntax. The exercises are also a good way to get more comfortable with this stuff.

**Note:** The next tutorial talks about the `meta` primary in detail, so we recommend that you ignore it until then.

# Exercises
1. Given `find docker/containers`, what is the depth of each entry? Hint: Remember that depth starts from `0` relative to the passed-in paths.

    1. `docker/containers`

        {% include exercise_answer.html answer="<code>0</code>" %}

    1. `docker/containers/wash_tutorial_redis_1`

        {% include exercise_answer.html answer="<code>1</code>" %}

    1. `docker/containers/wash_tutorial_web_1/metadata.json`

        {% include exercise_answer.html answer="<code>2</code>" %}

    1. `docker/containers/wash_tutorial_redis_1/fs/var/log/apt`

       {% include exercise_answer.html answer="<code>5</code>" %}

1. Here’s the invocation we used to print out all containers that started with `wash_tutorial`:

        find docker/containers -maxdepth 1 -name 'wash_tutorial*'

   Note that we had to use the `maxdepth` option to prevent `find` from recursing into a given container.

   We can use the `kind` primary to simplify the above invocation. What would that look like? Hint: Type `find --help kind` to see the `kind` primary’s documentation. You should be able to adapt one of the existing examples. Also, is the `containers` part of `docker/containers` still necessary?

   {% include exercise_answer.html answer="<code>find docker -kind '*container' -name 'wash_tutorial*'</code> is the simplest possible invocation. Notice that the <code>kind</code> primary eliminated the <code>containers</code> part of the path, and also removed the <code>maxdepth</code> option." %}

   You might be wondering what is “simpler” about the slightly longer `find docker -kind '*container' -name 'wash_tutorial*'`. The answer is that it makes it clear in the invocation that the kind of entries being filtered on are Docker containers. That information is not obvious in the previous invocation thanks to the presence of the `maxdepth` option.

1. This exercise introduces you to some of the other primaries. You should try to provide an example that shows the given primary in action.

    1. What primary lets you filter on an entry's creation time?

        {% include exercise_answer.html answer="<code>crtime</code><br /><br /><code>find docker -k '*container' -crtime -24h</code> would give you all Docker containers that were created within the last 24 hours." %}

    1. What primary lets you filter on an entry's last modification time?
        {% include exercise_answer.html answer="<code>mtime</code><br /><br /><code>find docker/containers/wash_tutorial_redis_1/fs/var/log -name '*.log' -mtime -1h</code> would give you all <code>.log</code> files in the <code>wash_tutorial_redis_1</code> container's <code>/var/log</code> directory that were modified within the last hour." %}

    1. What primary lets you filter on an entry's size?
        {% include exercise_answer.html answer="<code>size</code><br /><br /><code>find docker/volumes/wash_tutorial_redis -size -1k</code> would give you all of the <code>wash_tutorial_redis</code> volume's files whose size is less than 1 kibibyte." %}

1.  This exercise is meant to make you more comfortable with `find`'s expression syntax. Your answer for each part should be a `find` invocation that answers the given question.

    1. What are all the files that start with `dpkg` or `history` in the `wash_tutorial_redis_1`'s `/var/log` directory? Hint: The start path should be `docker/containers/wash_tutorial_redis_1/fs/var/log`.
        {% include exercise_answer.html answer="<code>find docker/containers/wash_tutorial_redis_1/fs/var/log -name 'dpkg*' -o -name 'history*'</code>" %}

    1. What are all of the `.log` files in the `wash_tutorial_redis_1`'s `/var/log` directory that start with `dpkg`?
        {% include exercise_answer.html answer="<code>find docker/containers/wash_tutorial_redis_1/fs/var/log -name '*.log' -a -name 'dpkg*'</code>" %}

    1. What are all of the `.log` files in the `wash_tutorial_redis_1`'s `/var/log` directory that start with `dpkg` OR `history`? Hint: You can use `()` to override precedence rules.
        {% include exercise_answer.html answer="<code>find docker/containers/wash_tutorial_redis_1/fs/var/log -name '*.log' \( -name 'dpkg*' -o -name 'history*' \)</code>" %}

# Next steps

Now that you have a basic understanding of the `find` command, you can move on to learning about the [meta primary](meta-primary).
