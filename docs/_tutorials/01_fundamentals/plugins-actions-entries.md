---
title: Understanding plugins, actions, and entries
---
**Note:** Remember that these are _hands-on_ tutorials. Thus, you should be able to replicate all the commands and see similar results.

This tutorial introduces you to Wash, which is a shell environment layered on top of the system shell. You’ll learn about plugins, actions and entries while navigating through the Docker plugin.

First, let’s start up the Wash shell.

```
bash-3.2$ wash
Welcome to Wash!
  Wash includes several built-in commands: wexec, find, list, meta, tail.
  See commands run with wash via 'whistory', and logs with 'whistory <id>'.
Try 'help'
wash . ❯ ls
aws/
docker/
gcp/
kubernetes/
```

**Note:** You might see some warning messages about plugins failing to load. If you are not planning on using those plugins, then feel free to exclude them from the [list of loaded plugins](../../docs#config). Otherwise, follow the suggestions in the warning messages to properly setup the remaining plugins (and remember to restart the Wash shell!).

Each thing listed here is a ‘directory’. Each 'directory' is a Wash plugin, and each Wash plugin serves as an adapter between a given vendor and Wash. Plugins are the sole mode of interaction between a Wash user and a given vendor's API. For example, the `docker` directory lets you interact with Docker resources like containers and volumes. Similarly, the `aws` directory lets you interact with AWS resources like EC2 instances and S3 objects.

Everything in Wash is an entry, including resources. Each plugin provides a hierarchical view of all its entries. With Wash, you navigate through a plugin's API in the same way that you would navigate the Linux command line -- by changing directories, listing the contents of a directory, and performing actions. 

Each entry supports a specific set of Wash actions.  For example, you could use `list` to list  an entry's children, `exec` to execute a command on the entry, or `read` to read an entry's contents. Note that entries which support `list` are represented as directories in the shell. All other entries are represented as files.

Let's walk through an example to see what all of this looks like in practice. We'll navigate through the Docker plugin and interact with some of its entries.

```
wash . ❯ cd docker
wash docker ❯
```

Notice how the prompt changed from `wash . >` to `wash docker >`. This is a useful way to track your current location when navigating through a given plugin.

```
wash docker ❯ ls
containers/
volumes/
```

From the output, we see that the Docker plugin lets us interact with containers and volumes via the `containers` and `volumes` entries. These entries support the `list` action, so they're represented as directories. Let’s try interacting with some containers.

```
wash docker ❯ cd containers
wash docker/containers ❯ ls
wash_tutorial_redis_1/
wash_tutorial_web_1/
```

Note that all the entries listed here are Docker containers. We can use the `winfo` command to see a Docker container’s supported actions.

```
wash docker/containers ❯ winfo wash_tutorial_redis_1/
Path: /Users/enis.inan/Library/Caches/wash/mnt050787757/docker/containers/wash_tutorial_redis_1
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

It looks like Docker containers support the `exec` action. That means we can execute commands on them. Let's find out some information about `wash_tutorial_redis_1`.

```
wash docker/containers ❯ wexec wash_tutorial_redis_1 uname
Linux
```

Nice! Also notice that the Docker container supports the `list` action. That means it’s modeled as a directory, so we can `cd` into it.

```
wash docker/containers ❯ cd wash_tutorial_redis_1
wash docker/containers/wash_tutorial_redis_1 ❯ ls
fs/
log
metadata.json
```

Everything listed here is specific to the `wash_tutorial_redis_1` container. For example, `log` represents the `wash_tutorial_redis_1` container’s log. Let's look at its supported actions with `wsinfo`:

```
wash docker/containers/wash_tutorial_redis_1 ❯ winfo log
Path: /Users/enis.inan/Library/Caches/wash/mnt947236113/docker/containers/wash_tutorial_redis_1/log
Name: log
CName: log
Actions:
- read
- stream
Attributes: {}
```

Notice that `log` doesn’t support the `list` action. That means it’s represented as a file, so if we try to `cd` into it, the action fails:

```
wash docker/containers/wash_tutorial_redis_1 ❯ cd log
cd: not a directory: log
```

However, `log` does support `read` and `stream`, so we can `cat` and `tail` its contents.

```
wash docker/containers/wash_tutorial_redis_1 ❯ cat log
1:C 04 Oct 2019 22:30:06.952 # oO0OoO0OoO0Oo Redis is starting oO0OoO0OoO0Oo
1:C 04 Oct 2019 22:30:06.952 # Redis version=5.0.6, bits=64, commit=00000000, modified=0, pid=1, just started
1:C 04 Oct 2019 22:30:06.952 # Configuration loaded
1:M 04 Oct 2019 22:30:06.954 * Running mode=standalone, port=6379.
1:M 04 Oct 2019 22:30:06.954 # WARNING: The TCP backlog setting of 511 cannot be enforced because /proc/sys/net/core/somaxconn is set to the lower value of 128.
1:M 04 Oct 2019 22:30:06.954 # Server initialized
1:M 04 Oct 2019 22:30:06.954 # WARNING you have Transparent Huge Pages (THP) support enabled in your kernel. This will create latency and memory usage issues with Redis. To fix this issue run the command 'echo never > /sys/kernel/mm/transparent_hugepage/enabled' as root, and add it to your /etc/rc.local in order to retain the setting after a reboot. Redis must be restarted after THP is disabled.
1:M 04 Oct 2019 22:30:06.954 * Ready to accept connections
wash docker/containers/wash_tutorial_redis_1 ❯ tail -f log
===> log <===
1:C 04 Oct 2019 22:30:06.952 # oO0OoO0OoO0Oo Redis is starting oO0OoO0OoO0Oo
1:C 04 Oct 2019 22:30:06.952 # Redis version=5.0.6, bits=64, commit=00000000, modified=0, pid=1, just started
1:C 04 Oct 2019 22:30:06.952 # Configuration loaded
1:M 04 Oct 2019 22:30:06.954 * Running mode=standalone, port=6379.
1:M 04 Oct 2019 22:30:06.954 # WARNING: The TCP backlog setting of 511 cannot be enforced because /proc/sys/net/core/somaxconn is set to the lower value of 128.
1:M 04 Oct 2019 22:30:06.954 # Server initialized
1:M 04 Oct 2019 22:30:06.954 # WARNING you have Transparent Huge Pages (THP) support enabled in your kernel. This will create latency and memory usage issues with Redis. To fix this issue run the command 'echo never > /sys/kernel/mm/transparent_hugepage/enabled' as root, and add it to your /etc/rc.local in order to retain the setting after a reboot. Redis must be restarted after THP is disabled.
1:M 04 Oct 2019 22:30:06.954 * Ready to accept connections
^C
```

(Hit `Ctrl+C` to cancel `tail -f`).

That’s enough navigation for now. Let’s go back to the Wash root.

```
wash docker/containers/wash_tutorial_redis_1 ❯ cd $W
wash . ❯
```

Notice that our prompt changed back to `wash . >`. That means we are indeed back at the Wash root. The `W` environment variable stores the Wash root’s absolute path, so you can invoke `cd $W` anytime you want to go back to the Wash root.

# Exercises

{% include exercise_reminder.md %}

1. You can tab-complete entries! Try using tab-completion to type `ls docker/containers/wash_tutorial_web_1`.

1. You can also glob entries! The following parts are meant to show you some interesting things that you can do with globbing.

    1. What’s the output of each command? Try to give a high-level overview. For example, something like _Prints out all the plugins_ is an acceptable answer for the command `echo *`. A more specific answer like _Prints out aws, docker, kubernetes, and gcp_ is also OK.

        1. `echo docker/*`
        2. `echo docker/containers/*`
        3. `echo docker/containers/wash_tutorial*`
        4. `echo docker/containers/*redis*`
        5. `echo docker/volumes/*`

        {% capture answer_2a %}
          <ol>
            <li>The containers and volumes directories.</li>
            <li>All Docker containers.</li>
            <li>All Docker containers that start with <code>wash_tutorial</code></li>
            <li>All Docker containers that contain the <code>redis</code> string</li>
            <li>All Docker volumes</li>
          </ol>
        {% endcapture %}
        {% include exercise_answer.html answer=answer_2a %}

    1. How would you tail every container’s log file (the `log` entry)? Hint: The invocation is of the form `tail -f <glob>`.

        {% include exercise_answer.html answer="<code>tail -f docker/containers/*/log</code>" %}

1. This exercise is broken up into several parts.

    1. We saw three entries when we `ls`’ed a Docker container: `log`, `fs`, and `metadata.json`. We already know that the `log` entry represents the container’s log. What do you think `fs` represents? Hint: Try `cd`’ing into it and `ls`’ing stuff. Use the `wash_tutorial_redis_1` container.

        {% include exercise_answer.html answer="<code>fs</code> represents the root directory of the container’s filesystem. It lets you navigate through the container as if you were logged onto it via something like SSH. As you’ll soon see, this lets you do some pretty cool stuff." %}

    1. Inside the `wash_tutorial_redis_1` directory, what command lets you read its `/var/log/apt/history.log` file? What command lets you tail it? Hint: `cat` lets you read a file. `tail -f` lets you tail it.

        {% include exercise_answer.html answer="<code>cat fs/var/log/apt/history.log</code> lets you read the <code>/var/log/apt/history.log</code> file. <code>tail -f fs/var/log/apt/history.log</code> lets you tail the <code>/var/log/apt/history.log</code> file. Thus, you can read/tail a Docker container’s log files as if you were logged onto it." %}

    1. What command lets you tail every container’s `/var/log/apt/history.log` file? Hint: See Exercise 2b’s answer.

        {% include exercise_answer.html answer="<code>tail -f docker/containers/*/fs/var/log/apt/history.log</code>. Thus, you can tail log files on multiple containers." %}

    1. Again inside the `wash_tutorial_redis_1` directory, what command lets you tail every file with the `.log` extension in its `/var/log` directory? Hint: The glob `**/*.log` matches every file with the `.log` extension, including subdirectories.

        {% include exercise_answer.html answer="<code>tail -f fs/var/log/**/*.log</code>. This exercise is meant to remind you that everything in Wash is an entry, including a container’s files and directories. That means you can still glob them just like you would if you were logged onto the container." %}

1. This exercise asks you some questions about the AWS plugin. Try `cd`'ing and `ls`'ing through it to answer them.

    1. How would you read an S3 object's content? Assume you've `cd`'ed into its bucket.

       {% include exercise_answer.html answer="<code>cat &lt;object_key&gt;</code>" %}

    1. How would you read an EC2 instance's console output? Assume you've `cd`'ed into it.

       {% include exercise_answer.html answer="<code>cat console.out</code>" %}

    1. How would you read a specific file on an EC2 instance? How would you tail it? Assume you've `cd`'ed into the EC2 instance. Hint: See the answers to Exercise 3a and 3b.

       {% include exercise_answer.html answer="<code>cat fs/&lt;file_path&gt;</code> reads the file. <code>tail -f fs/&lt;file_path&gt;</code> tails the file." %}

    1. How would you exec the `uname` command on an EC2 instance? Assume you've `cd`'ed into a directory that contains EC2 instances. **Note:** There are some subtleties when exec'ing commands on EC2 instances. Make sure to read the output of <code>docs &lt;ec2_instance&gt;</code> before verifying your answer. And if you can't get exec to work, then please let us know on Slack!

       {% include exercise_answer.html answer="<code>wexec &lt;ec2_instance&gt; uname</code>" %}

1. This exercise asks you some questions about the GCP plugin. Try `cd`'ing and `ls`'ing through it to answer them.

    1. How would you read a storage object's content? Assume you've `cd`'ed into its bucket.

       {% include exercise_answer.html answer="<code>cat &lt;object_key&gt;</code>" %}

    1. How would you read a compute instance's console output? Assume you've `cd`'ed into it.

       {% include exercise_answer.html answer="<code>cat console.out</code>" %}

    1. How would you read a specific file on a compute instance? How would you tail it? Assume you've `cd`'ed into the compute instance. Hint: See the answers to Exercise 3a and 3b.

       {% include exercise_answer.html answer="<code>cat fs/&lt;file_path&gt;</code> reads the file. <code>tail -f fs/&lt;file_path&gt;</code> tails the file." %}

    1. How would you exec the `uname` command on a compute instance? Assume you've `cd`'ed into a directory that contains compute instances. **Note:** There are some subtleties when exec'ing commands on compute instances. Make sure to read the output of <code>docs &lt;compute_instance&gt;</code> before verifying your answer. And if you can't get exec to work, then please let us know on Slack!

       {% include exercise_answer.html answer="<code>wexec &lt;compute_instance&gt; uname</code>" %}

1. This exercise asks you some questions about the Kubernetes plugin. Try `cd`'ing and `ls`'ing through it to answer them.

    1. How would you list a namespace's pods? Assume you've `cd`'ed into the namespace.

       {% include exercise_answer.html answer="<code>ls pods</code>" %}

    1. How would you exec the `uname` command on a pod? Assume you've `cd`'ed into a directory that contains pods.

       {% include exercise_answer.html answer="<code>wexec &lt;pod&gt; uname</code>" %}

# Next steps

Now that you've learned about plugins, actions and entries, you can move on to learning about [attributes and metadata](attributes-metadata)

# Related Links

* Check out the [action docs]({{ '/docs#actions' | relative_url }}) to see all of Wash's available actions.
