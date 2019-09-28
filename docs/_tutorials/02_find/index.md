---
title: Filtering entries with find
main: true
sections:
  - title: 'Understanding depth, primaries and expression syntax'
    endpoint: 'depth-primaries-expression'

  - title: 'Metadata filtering with the meta primary'
    endpoint: 'meta-primary'
---
The following tutorials introduce you to the `find` command, which lets you filter on entries that satisfy a certain set of criteria. For example, you can use find to:
* Filter all Docker containers, Kubernetes pods, S3 buckets, and S3 objects that were created or modified in the last day.
* Filter all Docker containers, Kubernetes pods, AWS EC2 instances, and GCP compute instances that contain a specific label or tag (and also filter on that label or tag’s value).
* Filter all Docker containers that were built from a given image ID.
* Filter all AWS EC2 instances associated with a particular VPC ID.

And if you’re using any external plugins, then you can use `find` to filter on those entries as well. For example, if you’re using the `puppetwash` plugin, then `find` lets you filter nodes based on their fact values or on their report timestamps.

Once you’ve mastered the `find` command, you'll be able to filter on almost any desirable property of a given entry. Let's [get started](/depth-primaries-expression)!
