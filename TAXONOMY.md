# Taxonomy of Resources

This is an attempt to construct a taxonomy for cloud resources. We start by defining different protocols we expect resources may conform to, then define some common types of resources that support a combination of protocols.

## Protocols
- metadata: everything has metadata
- traversal (ls, find, tree): this object has children that can be enumerated
- content (cat, tail/streaming): this object has content we can view
- editable (edit): this object has content we can modify
- exec (bolt): we can invoke commands on this object

## Types

### Cloud Resources

#### Compute

Systems that provide compute capacity.

Protocols: metadata, content (logs), exec. Could arguably have filesystem traversal, but that's probably not feasible.

Examples: AWS EC2, AWS Lambda, K8s Pod, network device (with shell access)

#### Volume

A place to store hierarchical data. Volumes only contain filesystem resources.

Protocols: metadata, content (access logs, optional), traversal

Examples: AWS EBS, K8s Volume

#### Service

An appliance that provides a service.

Protocols: metadata (configuration), content (logs, optional)

Examples: AWS ELB, K8s Ingress, network device (with no shell access)

#### Grouping

Organizational tools for grouping resources. Groupings only contain cloud resources.

Protocols: metadata, traversal

Examples: user, K8s namespace, cloud service (such as K8s cluster, aws), region, K8s deployment, AWS group or tag

### Filesystem Resources

#### File

Protocols: metadata, content, editable

#### Directory

Protocols: metadata, traversal

#### Symbolic Link

Protocols: metadata

