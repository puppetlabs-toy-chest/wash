---
title: Viewing documentation with stree and docs
---
This tutorial covers the `stree` and `docs` commands, which are useful for viewing plugin-specific documentation.

# stree
You can use the `stree` command to get a high-level overview of an entry’s hierarchy. Using `stree` on a plugin root is especially useful, because it shows you the kinds of entries you can interact with, and how those entries are organized. Let’s look at an example.

```
wash . ❯ stree docker
docker
├── containers
│   └── [container]
│       ├── log
│       ├── metadata.json
│       └── fs
│           ├── [dir]
│           │   ├── [dir]
│           │   └── [file]
│           └── [file]
└── volumes
    └── [volume]
        ├── [dir]
        │   ├── [dir]
        │   └── [file]
        └── [file]
```

Here’s a systematic way to read the above output.
* `ls`’ing the `docker` directory yields two directories: `containers` and `volumes`.
    * `ls`’ing the `containers` directory yields a bunch of Docker containers
        * `ls`’ing a Docker container yields a `log` entry representing the container’s log, a `metadata.json` file representing the container’s metadata, and an `fs` directory representing the root of the container’s filesystem.
            * `ls`’ing the `fs` directory is equivalent to `ls`’ing the container’s root.
                * `ls`’ing a directory in the container’s filesystem will yield more files and directories.
    * `ls`’ing the `volumes` directory yields a bunch of Docker volumes.
        * `ls`’ing a Docker volume yields its files and directories.
        * `ls`’ing a directory inside a Docker volume yields more files and directories.

As you can see, the Docker plugin lets us interact with Docker containers and volumes, including a specific container’s log and its filesystem. This is consistent with what we learned in the _Understanding plugins, actions and entries_ tutorial. However in that tutorial, we  arrived at this same set of information in a different way -- by `cd`’ing and `ls`’ing stuff in the Docker plugin.

**Note:** `<label>` implies a _singleton_ `<label>` entry while `[<label>]` implies zero or more `<label>` entries. For example, `containers` means there will only ever be one instance of a `containers` entry. `[container]` means there could be zero or more `container` entries.

Remember that `stree` can be used to get a high-level overview of any entry’s hierarchy (where applicable). For example,

```
wash . ❯ stree docker/containers/wash_tutorial_redis_1
docker/containers/wash_tutorial_redis_1
├── log
├── metadata.json
└── fs
    ├── [dir]
    │   ├── [dir]
    │   └── [file]
    └── [file]
```

which matches what was under the `[container]` node in the previous example.

## Exercises
In the following exercises, try to answer the questions using only the `stree` command. {% include exercise_reminder.md %}

1. This exercise will ask you questions about the AWS plugin.

    1. What would you get if you `ls`’ed the AWS directory?

       {% include exercise_answer.html answer="A bunch of AWS profiles." %}

    1. What would you get if you `ls`’ed an AWS profile?

       {% include exercise_answer.html answer="A <code>resources</code> directory." %}

    1. What would you get if you `ls`’ed a `resources` directory?

       {% include exercise_answer.html answer="<code>s3</code> and <code>ec2</code> directories." %}

    1. What would you get if you `ls`’ed a profile’s `ec2/instances` directory?

       {% include exercise_answer.html answer="All of the EC2 instances that are accessible by the given AWS profile." %}

    1. What would you get if you `ls`’ed a profile’s `s3/buckets` directory?

       {% include exercise_answer.html answer="All of the S3 buckets that are accessible by the given AWS profile." %}

    1. What would you get if you `ls`’ed an S3 bucket?

       {% include exercise_answer.html answer="All of the bucket’s objects organized in a hierarchical fashion. In other words, the bucket’s objects are organized like files and directories." %}

    1. Using your answers in parts (a) - (f), what are some of the resources that the AWS plugin lets you interact with?

       {% include exercise_answer.html answer="EC2 instances, S3 buckets and objects." %}

1. What kinds of entries does a GCP compute instance contain? Hint: `console.out` is short for console output.

   {% include exercise_answer.html answer="A GCP compute instance contains its console output via the <code>console.out</code> file, its metadata via the <code>metadata.json</code> file, and the root of its file system via the <code>fs</code> directory." %}

# docs
You can use the `docs` command to view an entry’s documentation. This is useful for answering questions like:
* What do I need to do to get the AWS plugin working?
* What does this `fs` entry represent?
* How do I get `exec` working on an AWS EC2 instance?
* What are the side effects of a particular exec on a GCP compute instance?
* How does the AWS plugin organize my S3 objects?

Here is an example of `docs`'s output for an `fs` entry:

```
wash . ❯ docs docker/containers/wash_tutorial_redis_1/fs
This represents the root directory of a container/VM. It lets you navigate
and interact with that container/VM's filesystem as if you were logged into
it. Thus, you're able to do things like 'cat'/'tail' that container/VM's files
(or even multiple files spread out across multiple containers/VMs).

Note that Wash will exec a command on the container/VM whenever it invokes a
List/Read/Stream action on a directory/file, and the action's result is not
currently cached. For List, that command is 'find -exec stat'. For Read, that
command is 'cat'. For Stream, that command is 'tail -f'.
```

## Exercises
{% include exercise_reminder.md %}

1. How does Wash list a Docker volume’s files and directories?

   {% include exercise_answer.html answer="It creates a temporary container, runs <code>find -exec stat</code> on it, then parses its output." %}

1. How does the AWS plugin detect a user’s AWS profiles? Hint: This type of information should be contained in the plugin root’s documentation.

   {% include exercise_answer.html answer="It reads the <code>AWS_SHARED_CREDENTIALS_FILE</code> environment variable or <code>$HOME/.aws/credentials</code> and <code>AWS_CONFIG_FILE</code> environment variable or <code>$HOME/.aws/config</code>" %}

1. How does the AWS (GCP) plugin organize an S3 (Storage) bucket’s objects? Hint: This type of information should be included in the bucket's documentation.

   {% include exercise_answer.html answer="It groups keys with common prefixes into directories. So, the objects <code>foo/bar</code>, <code>foo/baz</code> would be grouped under a <code>foo</code> directory." %}

# Caveats
`stree` and `docs` only work for entries with schemas. Thus, there is no guarantee that `stree` and `docs` will work for _every_ external plugin since entry schemas are optional. See the [entry schema docs]({{ '/docs/external-plugins#entry-schemas' | relative_url }}) to learn more about entry schemas.

# Next steps

That's the end of the _Fundamentals_ series! If you'd like to learn more about Wash, but don't know which tutorials to look at next, then check out the [tutorials](../) page for some ideas.
