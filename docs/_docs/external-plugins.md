---
title: External Plugins
---

* [Adding an external plugin](#adding-an-external-plugin)
* [Example Plugins](#example-plugins)
* [Libraries](#libraries)
* [Calling conventions](#calling-conventions)
  * [init](#init)
    * [Examples](#examples)
  * [list](#list)
    * [Examples](#examples-1)
    * [Method Tuples](#method-tuples)
  * [read](#read)
    * [Examples](#examples-2)
    * [Method Tuples](#method-tuples-1)
  * [write](#write)
    * [Examples](#examples-3)
  * [metadata](#metadata)
    * [Examples](#examples-4)
  * [stream](#stream)
    * [Examples](#examples-5)
  * [exec](#exec)
    * [Examples](#examples-6)
    * [Method Tuples](#method-tuples-2)
  * [schema](#schema)
    * [Examples](#examples-7)
    * [Method Tuples](#method-tuples-3)
  * [delete](#delete)
    * [Examples](#examples-8)
  * [signal](#signal)
    * [Examples](#examples-9)
  * [Entry JSON object](#entry-json-object)
  * [Entry schema graph JSON object](#entry-schema-graph-json-object)
  * [Errors](#errors)
* [Entry schemas](#entry-schemas)

# Adding an external plugin
Add the plugin to your `wash.yaml` file under the `external-plugins` key, and specify the _absolute_ path to the plugin script. An example `wash.yaml` config adding the `puppetwash` plugin is shown below:


```
external-plugins:
    - script: '/Users/enis.inan/GitHub/puppetwash/puppetwash.rb'
```

**Note:** You'll need to restart the Wash shell to enable any new plugins.

# Example Plugins

* [Boltwash](https://github.com/puppetlabs/boltwash) - view your Puppet Bolt inventory and explore target filesystems
* [Puppetwash](https://github.com/puppetlabs/puppetwash) - view your Puppet (Enterprise) instances and information about the managed nodes
* [Washhub](https://github.com/timidri/washhub) - navigate all your GitHub repositories at once without having to clone them
* [AWS IoT](https://gitlab.com/nwops/wash-iot) - view your AWS IoT devices and their shadow data
* [Washreads](https://github.com/MikaelSmith/washreads) - view your Goodreads bookshelves, also gives an intro into how to create a plugin
* [Spotify](https://github.com/binford2k/wash-spotifyfs) - view your Spotify playlists and tracks

# Libraries

* [Wash gem](https://github.com/puppetlabs/wash-ruby)

# Calling conventions
This section illustrates the calling conventions for each plugin script invocation. All calling conventions have the following general format

```
<plugin_script> <method> <path> <state> <args...>
```
where

* `<method>` is the Wash method that’s being invoked. This includes Wash actions like `list` and `exec`, and also non-Wash actions like `schema` and `metadata`.

* `<path>` is the entry’s filesystem path rooted at Wash’s mountpoint. For example, `/my_plugin` would be the `my_plugin`'s plugin root. `/my_plugin/foo` would be the `foo` child of the `my_plugin` entry.

* `<state>` consists of the minimum amount of information required to reconstruct the entry inside the plugin. It can be _any_ string. For example, `'{"klass": "Root"}'` could be a JSON object representing the plugin root in a higher-level programming language like Python or Ruby.

* `<args...>` are `<method>`’s arguments. For example if `<method>` is exec, then the exec’ed command would be included in `<args...>`.

Now let `<entry> = <path> <state>`. Then the plugin script’s usage becomes

```
<plugin_script> <method> <entry> <args...>
```

From this usage, we see that `<path>` and `<state>` are two different representations of an entry. `<path>` is useful for simple plugins where reconstructing the entry is easy. `<state>` is useful for more complicated plugins where entries could be represented as classes. For simplicity, we recommend that you use `<path>` or `<state>` to reconstruct your entries, but not both.

The remaining sections describe all the possible Wash methods that can be passed-in, including their calling and error conventions, and the expected results.

**Note:** Plugin script invocations run in their own process group (pgrp). Wash will send a `SIGTERM` signal to the pgrp on a cancelled API/filesystem request. If after five seconds the invocation process has not terminated, then Wash will send a `SIGKILL` signal.

**Note:** Unless otherwise mentioned, assume that all methods adopt the error conventions outlined in the [Errors](#errors) section.

## init
```
<plugin_script> init <config>
```

The `init` method is special. It is invoked only once, when the external plugin is loaded. `<config>` is JSON containing any config supplied to Wash under the plugin's key.

When `init` is invoked, the script must output an [entry JSON object](#entry-json-object) representing the plugin root. The *minimum* amount of information required for Wash to construct the plugin root is an empty object, `{}`.

**Note:** The plugin root's name _must_ match the basename of the plugin script (without the extension). For example, if the plugin script's path is `/path/to/myplugin.rb`, then the plugin root's name must be `myplugin`.

**Note:** Plugin roots _must_ implement `list`.

### Examples
Without config

```
bash-3.2$ /path/to/myplugin.rb init \{}
{}
```

With config

```yaml
external-plugins:
  - script: '/path/to/myplugin.rb'
myplugin:
  profiles:
    - profile_a
    - profile_b
```

```s
bash-3.2$ /path/to/myplugin.rb init '{"profiles":["profile_a","profile_b"]}'
{}
```

## list
`<plugin_script> list <path> <state>`

When `list` is invoked, the script must output an array of [entry JSON objects](#entry-json-object).

### Examples

```
bash-3.2$ /path/to/myplugin.rb list /myplugin/foo ''
[
  {
    "name": "bar",
    "methods": [
      "list",
      "exec"
    ]
  },
  {
    "name": "baz",
    "methods": [
      "read",
      "stream"
    ]
  }
]
```

### Method Tuples

`list`'s method tuple value represents a prefetched entry. The output would be an array of other entries, identical to the output from running the script in the previous examples.

Additionally Wash supports _core entries_ that provide built-in functionality that plugins can take advantage of.

_core entries_ can be used by returning an entry object with `type_id` set to the _core entries_ name surrounded by double underscores (`__core::entry__`) and the `name` field to identify the entry. You can also specify options for a _core entry_ using the `state` field containing serialized JSON of the options. If your plugin use _schemas_, then when using a _core entry_ you must still specify the _core entry_ as a child of the entry that lists it in the schema by it's `type_id`.

* `volume::fs`: a representation of your entry's filesystem that uses its `exec` method to access it. The `os.login_shell` attribute is used to determine how to interact with the filesystem; if not set it assumes `posixshell`. _Options_:
  * `maxdepth`: identifies how many levels of filesystem to fetch in a single batch to support trade-offs between `exec` latency and file density in the volume.

**EXAMPLES**
```
[
  "list",
  [
    {
      "type_id": "__volume::fs__",
      "name": "fs",
      "state": "[\"maxdepth\": 2]"
    },
    ...
  ]
]
```

A corresponding schema for the listing entry (given `type_id: example`) would be
```
"example": {
  "label": "example",
  "methods": [
    "list",
    "exec",
    "schema"
  ],
  "children": [
    "__volume::fs__",
    ...
  ]
}
```

## read
The default calling convention for `read` is

```
<plugin_script> read <path> <state>
```

which should output the entry's content.

If the plugin's API lets you read the entry's content in blocks, then you should implement the block-readable calling convention instead

```
<plugin_script> read <path> <state> <size> <offset>
```

which should output `<size>` bits of the entry's content starting at `<offset>`. Note that `<size>` and `<offset>` are 64-bit integers. You may assume valid input, i.e. that `0 <= <offset> < <size_attribute>` and that `0 <= <size> <= <size_attribute> - <offset>`.

### Examples

```
# Default signature
bash-3.2$ /path/to/myplugin.rb read /myplugin/foo ''
Some content
```

```
# Block-readable signature
bash-3.2$ /path/to/myplugin.rb read /myplugin/foo '' 3 0
Som
```

where `Some content` is the entry's content.

### Method Tuples

`read`'s method-tuple can be specified as either `["read", <string_value>]` or `["read", <block_readable?>]` (where `<block_readable?>` is a Boolean value).

If given a `<string_value>`, the string will be used as the result of a `read` instead of calling your plugin script, and the entry's `size` attribute will be set to the prefetched content size.

If the plugin implements the block-readable calling convention, then it must provide the `<block_readable?>` signature, as in `["read", true]`. Entries that implement `read`'s default signature _can_ specify `read` as the method-tuple `["read", false]`, but this is not required. 

## write
`<plugin_script> write <path> <state>`

When `write` is invoked, the script must read from `stdin` to get the content to write to the entry.

Wash distinguishes between two different patterns for things you can read and write. It considers a "file-like" entry to be one with a defined size (so the `size` attribute is set when listing the entry). Reading and writing a "file-like" entry edits the contents. The data passed to `stdin` is meant to be the entire content of the file.

Something that can be read and written but doesn't define size has different characteristics. Reading and writing are not symmetrical: if you write to it then read from it, you may not see what you just wrote. So these non-file-like entries error if you try to open them with a ReadWrite handle. If your plugin implements non-file-like write-semantics, remember to document how they work in the plugin schema's description.

### Examples

```
bash-3.2$ echo 'new content' | /path/to/myplugin.rb write /myplugin/foo ''
```

results in changing the entry's content to `new content`.

## metadata
`<plugin_script> metadata <path> <state>`

When `metadata` is invoked, the script must output a JSON object representing the entry's metadata.

**Note:** Only implement `metadata` if the entry has additional metadata properties that couldn't be included in the partial metadata because doing so would have slowed down parent#List.

### Examples

```
bash-3.2$ /path/to/myplugin.rb metadata /myplugin/foo ''
{
  "key1": "value1",
  "key2": "value2"
}
```

## stream
`<plugin_script> stream <path> <state>`

When `stream` is invoked, the first line of the script's output must contain the `200` header. This header tells Wash that the entry's data is about to the streamed. After it outputs the header, the script must then stream the entry's data. Wash will continue to poll `stdout` for any updates until either the streaming process exits, or the user cancels the request.

### Examples

```
bash-3.2$ /path/to/myplugin.rb stream /myplugin/foo ''
200
foo
bar
baz
...
```

where the `...` indicate indefinitely streaming content.

## exec
`<plugin_script> exec <path> <state> <opts> <cmd> <args...>`

where `<opts>` is the JSON serialization of the exec options. If the `input` key is included as part of `opts` in a request to the `exec` endpoint, then its content is passed-in as stdin to the plugin script and `opts["stdin"]` is set to `true`. Otherwise, `opts["stdin"]` is set to `false`.

When `exec` is invoked, the plugin script's `stdout` and `stderr` must be connected to `cmd`'s `stdout` and `stderr`, and it must exit the `exec` invocation with `cmd`'s exit code.

Because `exec` effectively hijacks `<plugin_script> exec` with `<cmd> <args...>`, there is currently no way for external plugins to report any `exec` errors to Wash. Thus, if `<plugin_script> exec` fails to exec `<cmd> <args...>` (e.g. due to a failed API call to trigger the exec), then that error output will be included as part of `<cmd> <args...>`'s output when running `wash exec`.

### Examples

```
bash-3.2$ /path/to/myplugin.rb exec /myplugin/foo '' '{"tty": true}' echo bar
bar
bash-3.2$ echo "$?"
0
```

### Method Tuples

`exec`'s tuple value represents an implementation of `exec`. Wash will use this implementation to handle all `exec` calls, so you do not have to implement `exec`'s plugin script invocation for this entry.

Currently, only implementations provided by the `transport` package are supported. The method tuple must be
```
[
  "exec",
  {
    "transport": <transport>,
    "options": { ... }
  }
]
```
where the object in the `options` field specifies transport-specific options. Supported transports are:

* `ssh`: connect using SSH. It will look up port, user, and other configuration by exact hostname match from default SSH config files (global SSH config is currently ignored). If present, a local SSH agent will be used for authentication. The known hosts file will be ignored if StrictHostKeyChecking=no in your SSH config. _Options_ (string values unless otherwise specified):
  * `host`: (required) the hostname to connect to
  * `port`: (integer) overrides port from SSH config (defaults to 22)
  * `user`: overrides user from SSH config
  * `fallback_user`: will be used if no user is specified in SSH config; if a user is not specified anywhere, will default to `root`
  * `password`: used for password-based authentication
  * `identity_file`: path to a private key file for public-key-based authentication
  * `known_hosts`: path to a known hosts file for server authentication
  * `host_key_alias`: can be used if the hostname specified in known hosts differs from `host`
  * `retries`: (integer) can be set to retry every 500ms for that many times

**EXAMPLES**
```
[
  "exec",
  {
    "transport": "ssh",
    "options": {
      "host": "example.com",
      "user": "ubuntu",
      "port": 2222
    }
  }
]
```

## schema
`<plugin_script> schema <path> <state>`

When `schema` is invoked, the script must output an [entry schema graph JSON object](#entry-schema-graph-json-object).

[Entry schemas](#entry-schemas) are an _on/off_ feature. If the plugin root implements `schema`, then entry schemas are _on_. Otherwise, entry schemas are _off_. If entry schemas are _on_, then Wash will require all subsequent entries to implement `schema` and to include a `type_id` key (including the root). Wash will return an error if both these conditions aren't met. If entry schemas are _off_, then Wash will return an error if any subsequent entry implements `schema`. The latter restriction's necessary to ensure consistent behavior across your plugin.

### Examples
```
bash-3.2$ /path/to/myplugin.rb schema /myplugin/foo ''
{
  "foo_type_id": {
    "label": "foo_label",
    "methods": [
      "list"
    ]
  }
}
```

### Method Tuples

`schema`s method tuple value represents a prefetched result. However, only the root is allowed to do this. Thus, if any other entry attempts to prefetch its schema, Wash will return an error.

## delete
`<plugin_script> delete <path> <state>`

When `delete` is invoked, the script must output a boolean JSON. `true` means that the entry was deleted. `false` means that the entry is marked for deletion and will eventually be deleted by the plugin's API.

`delete` should ensure that both the entry and its children are removed. If the entry has any dependencies that need to be deleted, then `delete` should error.

**Note:** If you anticipate `delete` taking a long time (> 30 seconds), then output `false`.

### Examples
```
bash-3.2$ /path/to/myplugin.rb delete /myplugin/foo ''
true
```

## signal
`<plugin_script> signal <path> <state> <signal>`

A successful `signal` invocation should return when the signal was successfully sent, and it should not output anything.

**Note:** `<signal>` is downcased. If entry schemas are enabled, then `<signal>` will be a valid signal.

**Note:** Checkout the [signal action docs]({{ 'docs/#signal' | relative_url }}) for a list of common signal names.

### Examples
```
bash-3.2$ /path/to/myplugin.rb signal /myplugin/foo '' start
bash-3.2$
```

## Entry JSON object
This section describes the JSON object representing a serialized entry. An entry JSON object supports the following keys. Only the `name` and `methods` keys are required.

* `name` is a string representing the entry's raw name.

* `methods` is an array specifying the entry's implemented methods. Each element in the array can be a string representing the method's name, or a method-tuple of `[<method_name>, <value>]`, where `value` is method-specific. See the _Tuple Value_ section of each method for specific details. The most common tuple value is a prefetched result, used in the examples below.

  **EXAMPLES**
  ```
  [
    "list",
    "exec"
  ]
  ```

  ```
  # With method tuples
  [
    ["list", [
      {
        "name": "foo",
        "methods": [
          ["read", "some content"],
          "stream"
        ]
      }
    ]],
    "exec"
  ]
  ```

  Notice that `list`'s `<value>` matches what's outputted by a `list` invocation. Similarly, `read`'s `<value>` matches what's outputted by a `read` invocation.

* `attributes` is an object specifying the entry's attributes. See the [attributes docs]({{ '/docs#attributes' | relative_url }}) for a list of all the supported Wash attributes.

  **EXAMPLES**
  ```
  {
    "mtime": 1551942012,
  }
  ```

* `partial_metadata` is an object specifying the entry's partial metadata. The attributes should be a subset of this.

  **EXAMPLES**
  ```
  {
    "foo_key": "foo_value",
  }
  ```

* `state` is a string specifying the entry's state. This is the same `<state>` that's passed into _all_ plugin script invocations.

* `cache_ttls` is an object that only supports the `list`, `read` and `metadata` keys (all other keys are ignored). Each key corresponds to a cached method. Their value represents the number of seconds that the method's result should be cached (`ttl` is short for time to live).

  **EXAMPLES**
  ```
  {
    "metadata": 10,
    "read": 20
  }
  ```

  Here, we see that Wash will cache this entry's `metadata` result for 10 seconds, and its `read` result for 20.

* `slash_replacer` is a single character that overrides the default slash replacer.

* `inaccessible_reason` is a string specifying why the entry is inaccessible. The current plugin configuration may not provide sufficient permissions to access a particular resource. Rather than triggering an error in Wash, this resource can be omitted when listing available resources. The `inaccessible_reason` attribute provides a place to flag that the resource should be omitted from list results and log a reason for its omission.

Below is an example entry JSON object showcasing all the possible keys at once.

```
{
  "name": "foo",
  "methods": [
    "list"
  ],
  "attributes": {
    "mtime": 1551942012
  },
  "partial_metadata": {
    "foo_key": "foo_value",
  }
  "state": "\"{\"klass\":\"Foo\"}\"",
  "cache_ttls": {
    "read": 10
  },
  "slash_replacer": ":",
  "inaccessible_reason": "permission denied"
}
```

## Entry schema graph JSON object
This section describes the JSON object representing a serialized entry schema graph. Its keys are type IDs, and its values are entry schema JSON objects. A type ID is the unique type-identifier of a specific kind of entry (typically the fully-qualified class-name). An entry schema JSON object represents that entry’s schema.

The entry schema JSON object supports the following keys. Only the `label` and `methods` keys are required. If the entry's a parent (implements `list`), then its child schemas must also be included via the `children` key.

* `label` is a string representing the entry's label. This is what's shown by the `stree` command.

* `methods` is an array of strings specifying the entry's implemented methods.

   **EXAMPLES**
   ```
   [
     "list",
     "exec"
   ]
   ```

* `singleton` is a boolean value that indicates whether the entry's a singleton.

* `description` is a string representing the entry's description. This is what's shown by the `docs` command. Note that you should always set a description for plugin roots. That description should contain just enough details for a user to get your plugin working. It should also contain any known issues related to your plugin.


* `children` is an array of strings specifying the type IDs of the entry's children.

   **EXAMPLES**
   ```
   [
     "foo_type_id",
     "bar_type_id"
   ]
   ```

* `signals` is an array of hashes specifying the entry's supported signals and signal groups.

  **EXAMPLES**
  ```
  [
    {
      "name": "start",
      "description": "Start the thing"
    },
    {
      "name": "stop",
      "description": "Stop the thing"
    },
    {
      "name": "linux",
      "description": "Consists of all the supported Linux signals like SIGHUP, SIGKILL",
      "regex": "\\Asig*"
    }
  ]
  ```

  Note that the [regex](https://golang.org/pkg/regexp/syntax/#pkg-overview) key describes a supported signal in the given signal group. It distinguishes signal groups from signals.

  A given signal is valid iff it matches a supported signal's _name_ OR a supported signal group's _regex_. See the [signal action docs]({{ '/docs#signal' | relative_url }}) for a list of common signal names. You should try to reuse these names where applicable.

* `partial_metadata_schema` is a serialized JSON schema representing the entry's `partial metadata` schema.

* `metadata_schema` is a serialized JSON schema representing the entry's `metadata` schema.

  **EXAMPLES**
  ```
  {
    "patternProperties": {
      ".*": {
        "type": "string"
      }
    },
    "type": "object"
  }
  ```

  Note that this JSON schema implies that the entry's `partial metadata` OR `metadata` can include any property of any type.

Below is an example entry schema JSON object showcasing all the possible keys at once.

```
{
  "label": "foo",
  "methods": [
    "list",
    "signal"
  ],
  "singleton": false,
  "description": "A description.",
  "children": [
    "bar_type_id"
  ],
  "partial_metadata_schema": {
    "patternProperties": {
      ".*": {
        "type": "string"
      }
    },
    "type": "object"
  },
  "metadata_schema": {
    "patternProperties": {
      ".*": {
        "type": "string"
      }
    },
    "type": "object"
  },
  "signals": [
    {
      "name": "start",
      "description": "Start the thing"
    }
  ]
}
```

Below is an example entry schema graph JSON object

```
{
  "foo_type_id": {
    "label": "foo",
    "methods": [
      "list"
    ],
    "children": [
      "bar_type_id"
    ]
  },
  "bar_type_id": {
    "label": "bar",
    "methods": [
      "read"
    ]
  }
}
```

## Errors
All errors are printed to `stderr`. A method invocation is said to have errored when the plugin script returns a non-zero exit code. In that case, Wash wraps all of `stderr` into an error object, then documents that error in the process' activity and the server logs.

**Note:** Not all method invocations adopt this error handling convention (e.g. `exec`). The error handling for these "snowflake" methods is described in their respective sections.

# Entry schemas

Entry schemas are a _optional_ type-level overview of your plugin's hierarchy. They enumerate the kinds of things your plugins can contain, including what those things look like. For example, a Docker container's schema would answer questions like:

* Can I create multiple Docker containers?
* What's in a Docker container's metadata?
* What Wash actions does a Docker container support?
* If I `ls` a Docker container, what do I get?

These questions can be generalized to any Wash entry.

Entry schemas are a useful way to document your plugin without having to maintain a README. They are also useful for optimizing `find`, especially when `find` is used for metadata filtering. Without entry schemas, for example, an EC2 instance query like `find aws -meta '.tags[?]' '.key' termination_date` would cause `find` to recurse into every entry in the `aws` plugin, including non-EC2 instance entries like S3 objects. With entry schemas, however, `find` would only recurse into those entries that will eventually lead to an EC2 instance. The latter is a significantly faster (and less expensive) operation, especially for large infrastructures.
