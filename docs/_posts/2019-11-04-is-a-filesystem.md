---
title: "Wash is a filesystem"
summary: "Demonstrates how Wash provides a filesystem to leverage the tools you already use"
author: Michael Smith
twitter: lasthemy
---

One of the core principles of Wash is that interacting with things like a filesystem is familiar and powerful. Files are easy to manipulate; every system comes with a suite of tools to do so. They have permanence and a fixed place in a hierarchy, so they're easy to find again. When building Wash we wanted to leverage that for some basic operations.

So Wash is built on a ([FUSE](https://en.wikipedia.org/wiki/Filesystem_in_Userspace)) filesystem. That means native tools just work with it.

You can use `ls` and `cd` to explore Wash's hierarchy:
```
wash . > ls
aws/
docker/
gcp/
kubernetes/
wash . > cd gcp && ls
Wash/
another-project/
wash . > cd Wash && ls
compute/
storage/
```

Other tools that interact with the filesystem also work
```
wash . > tree -I fs
.
├── compute
│   └── michael-test-instance
│       ├── console.out
│       └── metadata.json
└── storage
    └── some-wash-stuff
        ├── an\ example\ folder
        │   └── static.sh
        └── reaper.sh

5 directories, 4 files
```
> Note that I specifically excluded the `fs` directory (present in compute instances) because it would traverse the entire filesystem of the instance.

The hierarchy it presents is decided by plugins specific to each service; AWS, GCP, Docker, and Kubernetes are in the core application, and you can easily [add more](https://puppetlabs.github.io/wash/tutorials/03_extending_wash/). They tend to reflect the organization of resources in those services, e.g. GCP lists [projects](https://cloud.google.com/resource-manager/docs/creating-managing-projects), then within those the available resource types (compute and storage) and instances of those resources (compute instances and storage buckets).

Other tools that interact with files also just work. Try `stat`, `less`, or your favorite editor.
