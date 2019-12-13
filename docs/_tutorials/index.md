---
title: Tutorials
---
**Note:** Make sure you've [installed Wash]({{ '/getting_started' | relative_url }}) before going any further.

This series of hands-on tutorials introduces you to Wash. Each tutorial assumes that you have the [test environment set-up](00_test_environment), and that you have an understanding of Wash’s key abstractions. If you're unfamiliar with Wash, start with the [Fundamentals](01_fundamentals/plugins-actions-entries). You can complete the tutorials in any order. If you're not sure where to start, here are a few good options: 

* [Filtering entries with find](02_find) is useful if your day-to-day job consists of filtering a bunch of different things. Note that these things do not have to be Docker, Kubernetes, AWS, or GCP resources; they can be anything. And if you're interested in filtering anything, check out [Extending Wash](03_extending_wash).

* [Extending Wash](03_extending_wash) is useful if you think Wash’s abstractions would make it easier for you to manage something other than the shipped plugins. Examples include your Spotify playlists, Puppet nodes, internal company APIs, Goodreads books, IOT devices, remote GitHub repos, etc.

* [Debugging](04_debugging/whistory) is useful if you plan on doing some complicated stuff with Wash that would require you to do a lot of debugging if anything fails.

Some tutorials include optional exercises at the end of each section. The exercises are meant to
* Test your understanding of the material
* Level up your Wash “skills”
* Introduce you to some of Wash’s other features like
    * Tab-completion and globbing
    * Exposing an AWS S3 or GCP storage bucket’s objects as files and directories
    * Reading the contents of an AWS S3 or GCP storage object
    * Exploring a VM or container's filesystem as if you were logged onto it
    * Tailing log files spread out across multiple AWS EC2 instances, Docker containers, or GCP compute instances. They also include filtering on recently updated log files.
    * Filtering AWS EC2 instances, Docker containers, Kubernetes pods or GCP compute instances on their tags or labels.

Think of the exercises as supplemental tutorial material. Even if you don’t want to complete an exercise, skimming through it might unearth some really useful features.

Note that the exercises are not meant to trick you. They are intended to be straightforward and fun. If you find yourself feeling frustrated with a specific exercise or question, please let us know in the #wash channel on [Puppet's community Slack](https://puppetcommunity.slack.com/?redir=%2Fapp_redirect%3Fchannel%3Dwash). We’ll try to find a way to simplify the exercise, or even remove it if it does not add any value.

All exercises include their answers so you can double-check your work. Some exercises may have more than one acceptable answer. If you have any questions on an exercise’s solution, or on the correctness of your answer, then please let us know on the Slack channel.
