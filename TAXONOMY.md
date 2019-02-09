# Taxonomy of Resources

This is an attempt to construct a taxonomy for cloud resources. We start by defining different protocols we expect resources may conform to, then define some common types of resources that support a combination of protocols.

## Protocols
- metadata: everything has metadata
- group traversal (ls, find, tree): this object has Cloud Resources that can be enumerated
- file traversal (ls, find, tree): this object has Filesystem Resources that can be enumerated
- content (cat): this object has content we can view
- stream (tail -f/streaming): this content has a stream of data we can follow
- editable (edit): this object has content we can modify
- exec (bolt): we can invoke commands on this object

## Types

### Cloud Resources

#### Compute

Systems that provide compute capacity.

Protocols: metadata, stream (logs), exec. Could arguably have filesystem traversal, but that's probably not feasible.

Examples: AWS EC2, AWS Lambda, K8s Pod, network device (with shell access)

#### Volume

A place to store hierarchical data.

Protocols: metadata, stream (access logs, optional), file traversal

Examples: AWS EBS, K8s Volume

#### Service

An appliance that provides a service.

Protocols: metadata (configuration), stream (logs, optional)

Examples: AWS ELB, K8s Ingress, network device (with no shell access)

#### Group

Organizational tools for grouping resources.

Protocols: metadata, group traversal

Examples: user, K8s namespace, cloud service (such as K8s cluster, aws), region, K8s deployment, AWS group or tag

### Filesystem Resources

#### File

Protocols: attributes, content, editable

#### Directory

Protocols: attributes, file traversal
