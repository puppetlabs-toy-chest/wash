---
title: "Introducing Wash"
description: "Simplify exploring cloud infrastructure with Wash"
author: michaelsmith
---

Have you ever had to:

* List all your AWS EC2 instances or Kubernetes pods?
* Read/cat a GCP Compute instance's console output, or an AWS S3 object's content?
* Exec a command on a Kubernetes pod or GCP Compute Instance?
* Find all AWS EC2 instances with a particular tag, or Docker containers/Kubernetes pods/GCP Compute instances with a specific label?

If so, then some parts of the following tables might look familiar to you. If not, then here's how AWS/Docker/Kubernetes/GCP recommends that you do some of these tasks.

List all... | Command
----------------------|---
AWS EC2 instances     | `aws ec2 describe-instances --profile foo --query 'Reservations[].Instances[].InstanceId' --output text`
Docker containers     | `docker ps --all`
Kubernetes pods       | `kubectl get pods --all-namespaces`
GCP Compute instances | `gcloud compute instances list`

Read... | Command
--------------------------------------------|---
Console output of an EC2 instance           | `aws ec2 get-console-output --profile foo --instance-id bar`
Console output of a Google compute instance | `gcloud compute instances get-serial-port-output foo`
An S3 object's content                      | `aws s3api get-object content.txt --profile foo --bucket bar --key baz && cat content.txt && rm content.txt`
A GCP Storage object's content              | `gsutil cat gs://foo/bar`

Exec `uname` on... | Command
-----------------------------|---
An EC2 instance              | `ssh -i /path/my-key-pair.pem ec2-user@195.70.57.35 uname`
An a Docker container        | `docker exec foo uname`
Exec on a Kubernetes pod     | `kubectl exec foo uname`
On a Google Compute instance | `gcloud compute ssh foo --command uname`

Find by 'owner'... | Command
--------------------------|---
EC2 instances             | `aws ec2 describe-instances --profile foo --query 'Reservations[].Instances[].InstanceId' --filters Name=tag-key,Values=owner --output text`
Docker containers         | `docker ps --filter “label=owner”`
Kubernetes pods           | `kubectl get pods --all-namespaces --selector=owner`
Google Compute instance   | `gcloud compute instances list --filter=”labels.owner:*”`

From this, we see that you need to use different commands to List/Read/Exec/Find different things. Furthermore, these commands require you to install different applications that each come with their own set of (possibly conflicting) dependencies, and their own calling conventions. For example, to complete all the Find tasks specified in the table, you need to:

* Use the `aws ec2 describe-instances`, `docker ps`, `kubectl get pods`, `gcloud compute instances list` commands (4 different commands).

* Install the `aws`, `docker`, `kubectl` and `gcloud` applications (4 different applications). Note that `aws` and `gcloud` are Python applications, so you must also install Python. Also, `gcloud` only works with Python 2 so if you just have Python 3 installed on your machine, you must now get and install Python 2 and do the installation in such a way that it is easy for you to switch-back to using Python 3 for some of your other applications. This is not an easy thing to do, especially if you are not familiar with the Python ecosystem.

* Learn four different-but-similar DSLs for filtering stuff, which effectively means four different-but-similar ways of constructing and combining predicates on structured data (e.g. GCP's filter expressions, Kubernetes' field selectors, Kubernetes' label selectors, aws' describe-instances' --filters option, docker ps filtering, etc.).

That's a lot of stuff you have to use and learn to do some very fundamental and basic tasks. It naturally begs the question of whether there's a better way of performing these tasks that is (1) more expressive than what's presented here, and (2) does not require you to learn different commands to perform a task on different things.

The answer is Wash. With Wash, here's what the invocations look like:

List all... | Command
----------------------|---
AWS EC2 instances     | `find aws/foo -k '*ec2*instance'`
Docker containers     | `find docker -k '*container' `
Kubernetes pods       | `find kubernetes -k '*pod'`
GCP Compute instances | `find gcp -k '*compute*instance'`

Read... | Command
--------------------------------------------|---
Console output of an EC2 instance           | `cat aws/foo/resources/ec2/instances/bar/console.out`
Console output of a Google compute instance | `cat gcp/<project>/compute/foo/console.out`
An S3 object's content                      | `cat aws/foo/resources/s3/bar/baz`
A GCP Storage object's content              | `cat gcp/<project>/storage/foo/bar`

Exec `uname` on... | Command
-----------------------------|---
An EC2 instance              | `wexec aws/foo/resources/ec2/instances/bar uname`
An a Docker container        | `wexec docker/containers/foo uname`
Exec on a Kubernetes pod     | `wexec kubernetes/<context>/<namespace>/pods/foo uname`
On a Google Compute instance | `wexec gcp/<project>/compute/foo uname`

Find by 'owner'... | Command
------------------------|---
EC2 instances           | `find aws/foo -k '*ec2*instance' -meta '.tags[?].key' owner`
Docker containers       | `find docker -k '*container' -meta '.labels.owner' -exists`
Kubernetes pods         | `find kubernetes -k '*pod' -meta '.metadata.labels.owner' -exists`
Google Compute instance | `find gcp -k '*compute*instance' -meta '.labels.owner' -exists`

Contrast this with the commands you'd ordinarily have to use. We immediately see that using Wash means:

* The commands you use follow established, UNIX conventions.

* You no longer have to learn different commands to execute a task across different things. All you need is one command (`find` for List/Find; `cat` for Read; and `wexec` for Exec).

* You no longer have to install a bunch of different tools. All you need to install is the Wash binary.

* You no longer have to learn different DSLs for filtering stuff. All you need to learn is find's expression syntax and its individual primaries. Once you do that, you can filter on almost any conceivable property of your specific thing.

In fact, Wash is not just a centralization of basic and fundamental commands. It is a shell. It is a shell layered on top of an existing shell like Bash or ZSH. This means that:

* Everything is a file. Thus, interacting with a specific kind of thing like a Docker container is as easy as cd'ing into its containing “directory” and running a bunch of commands on the things in that “directory”. For example, something like `cd docker/containers` lets you follow-up with commands like `ls` (list all containers), `wexec foo uname` (execute the `uname` command on the foo container), or `tail foo/fs/var/log/messages` (tail the foo container's `/var/log/messages` file for updates).

* You can tab-complete and glob stuff (if your shell supports those features). For example, assuming you're in the `docker/containers` directory, typing in `ls f` then tab will work just the way you'd expect it to work if you were in a standard directory like `/var/log` (where f would be a regular file or directory). Similarly, `tail */fs/var/log/messages` will tail every container's `/var/log/messages` file for updates.[^1]

Furthermore, Wash is built on a plugin architecture. It ships with some default plugins for Docker, AWS, Kubernetes, and GCP, but you can easily extend it with your own plugin via the external plugin interface (and you can write that plugin in any language you want, including Bash). People have written external plugins for all sorts of things such as IoT devices, Goodreads, GitHub, PuppetDB, and even Spotify.

If this all sounds intriguing to you, please [download Wash]({{'getting_started' | relative_url}}) Wash and [give it a try]({{'tutorials' | relative_url}})!

[^1]: Notice that this example showcases another Wash feature: that you don't have to SSH into a bunch of different things to individually tail their logs. For example, something like `tail aws/foo/resources/ec2/instances/bar/fs/var/log/messages gcp/baz/compute/qux/fs/var/log/messages` will tail the `/var/log/messages` file on the bar EC2 instance and the qux GCP compute instance. As a fun example, the command `tail -f $(find -k 'aws/*ec2*instance' -o -k 'gcp/*compute*instance' | sed s:$:/fs/var/log/messages:g)` will tail the `/var/log/messages` file of every EC2 instance and GCP compute instance. (Note that the intermediate sed appends `/fs/var/log/messages` to the outputted paths. Baking this logic into find considerably slows things down. Also, `fs` is short for “filesystem”. It represents the root directory of the EC2 instance/GCP compute instance).
