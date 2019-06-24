+++
title= "External Plugins"
+++

- [Plugin Script](#Plugin-Script)
- [init](#init)
- [list](#list)
- [read](#read)
- [metadata](#metadata)
- [stream](#stream)
- [exec](#exec)
- [Errors](#Errors)
- [Aside (optional)](#Aside-optional)
- [Bash Example](#Bash-Example)

External plugins let Wash talk to other things outside of the built-in plugins. They can be written in any language. To write an external plugin, you need to do the following:

1. Write the [plugin script](#plugin-script). This is the script that Wash will shell out to whenever it needs to invoke a method on a specific entry within your plugin.

1. Add the plugin to your `wash.yaml` file under the `external-plugins` key, and specify the path to the plugin script. *The plugin's name is the basename of the script without the extension.* An example `wash.yaml` config adding the `sshfs` plugin is shown below:

    ```yaml
    external-plugins:
        - script: '/path/to/wash/website/static/docs/external_plugins/examples/sshfs.sh'
    ```

1. Start the Wash shell to see your plugin in action.

## Plugin Script

Wash shells out to the external plugin's script whenever it needs to invoke a method on one of its entries. The script must have the following usage:

```s
<plugin_script> <method> <path> <state> <args...>
```

where

* `<method>` is the Wash method that's being invoked
* `<path>` is the entry's filesystem path rooted at Wash's mountpoint. For example, `/<plugin_root_name>` would be the passed-in path for the plugin root.
* `<state>` consists of the minimum amount of information required to reconstruct the entry inside the plugin
* `<args...>` are the method's arguments.

`<path>` and `<state>` can be a bit confusing. To understand them, we recommend reading the [Aside](#aside), and to look at the provided [Bash](#bash-example) + Ruby external plugin examples to see how they're used. **TODO: Link a Ruby example**

The remaining sections describe all the possible Wash methods that can be passed-in, including their calling and error conventions, and the expected results.

NOTE: Plugin script invocations run in their own process group (pgrp). Wash will send a SIGTERM signal to the pgrp on a cancelled API/filesystem request. If after five seconds the invocation process has not terminated, then Wash will send a SIGKILL signal.

## init
The `init` method is special. It is invoked as `<plugin_script> init <config>`, and it is invoked only once, when the external plugin is loaded. `<config>` is JSON containing any config supplied to Wash under the plugin's key. Given a Wash config file (`wash.yaml`)

```yaml
external-plugins:
  - script: '/path/to/myplugin.rb'
myplugin:
  profiles:
    - profile_a
    - profile_b
```

the `init` method for a plugin named `myplugin` will be invoked with

```s
<plugin_script> init '{"profiles":["profile_a","profile_b"]}'
```

When `init` is invoked, the script must output a JSON object representing the plugin root. The *minimum* amount of information required for Wash to construct the plugin root is an empty object, `{}`.

You can include additional (optional) keys in the printed JSON object. These keys are:

* `methods`. This is an array specifying the list of methods, enumerated below, that can be called directly on the plugin entry. The plugin root must always include and implement the `list` method.
* `cache_ttls`. This specifies how many seconds each method's result should be cached (`ttl` is short for time to live). Currently, Wash caches the result of `list`, `read`, and `metadata`.
* `attributes`. This represents the entry's attributes (see the [`Attributes/Metadata`](/wash/docs#attributes-metadata) section). Time attributes are specified in Unix seconds. Octal modes must be prefixed with the `0` delimiter (e.g. like `0777`). Hexadecimal modes must be prefixed with the `0x` delimiter (e.g. like `0xabcd`).
* `slash_replacer`. This overrides the default slash replacer `#`.
* `state`. This corresponds to the `<state>` parameter in the plugin script's usage.

Below is an example JSON object showcasing all possible keys at once.

```json
{
  "methods": ["list"],
  "cache_ttls": {
    "list": 30
  },
  "attributes": {
    "mtime": 1551942012,
    "meta": {
      "LastModifiedTime": 1551942012,
      "Owner": "Wash",
    }
  },
  "slash_replacer": ":",
  "state": "{\"klass\":\"SSHFS::Directory\"}"
}
```

We see from `cache_ttls` that the result of `some_entry`'s `list` method will be cached for 30 seconds. We see from `attributes` that `some_entry` has an `mtime` attribute, and that it also includes the `meta` attribute. We see from `slash_replacer` that any `/`'es in the entry's returned name will be replaced by a `:` instead of a `#`. Finally, we see from `state` that `some_entry` has some state that Wash will pass-back in via the `<state>` parameter whenever it invokes one of its methods. In this case, `<state>` is a stringified JSON object containing the entry's class (`SSHFS::Directory`) in whatever language the plugin script was written in.

`init` adopts the standard error conventions described in the [Errors](#errors) section.

## list
`list` is invoked as `<plugin_script> list <path> <state>`. When `list` is invoked, the script must output an array of JSON objects. The *minimum* information required is each entry's name and its implemented methods

```json
{
  "name": "mydirectory",
  "methods": [
    "list"
  ]
}
```

The `<path>` takes the form of a UNIX-style path rooted at Wash's root directory. So the first call will be `<plugin_script> list /<plugin_name>`, followed by `<plugin_script> list /<plugin_name>/<directory_name>`, etc.

Each entry may additionally return any keys described in [init](#init).

Below is an example of valid `list` output:

```json
[
  {
    "name": "fooVM",
    "methods": [
      "list",
      "exec",
      "metadata"
    ],
    "attributes": {
      "mtime": 1558062927,
      "meta": {
        "LastModifiedTime": 1558062927,
        "Owner": "Alice"
      }
    },
    "state": "{\"klass\":\"SSHFS::VM\"}"
  },
  {
    "name": "barVM",
    "methods": [
      "list",
      "exec",
      "metadata"
    ],
    "attributes": {
      "mtime": 1558062927,
      "meta": {
        "LastModifiedTime": 1558062927,
        "Owner": "Alice"
      }
    },
    "state": "{\"klass\":\"SSHFS::VM\"}"
  }
]
```

If you're able to pre-fetch a method's result as part of the `list` method, then you can include the result as a tuple of `[<method>, <result>]` in the `methods` array. Pre-fetching is a useful way to avoid unnecessary plugin script invocations.

Below is an example that includes pre-fetched method results for a static directory that contains known files and content, but may also support streaming new updates dynamically (by invoking the `stream` method on the script). Notice how `list`'s result matches what would have been returned by `<plugin_script> list /<plugin_name>/mydir`. Note that when `read` content is provided in this manner, the size of that content will be automatically populated in `attributes`.

```json
[
  {
    "name": "mydir",
    "methods": [
      ["list", [
        {
          "name": "myfile 1",
          "methods": [
            ["read", "some content"],
            "stream"
          ]
        },
        {
          "name": "myfile 2",
          "methods": [
            ["read", "more content"],
            "stream"
          ]
        }
      ]]
    ]
  }
]
```

**NOTE:** Remember that the state displayed here is the same `<state>` parameter that will be passed in to other methods invoked on that entry. For example, if `read` is invoked on `fooVM` via `<plugin_script> read <parent_path>/fooVM <state>`, then the value of `fooVM`'s `state` key will be passed-in for `<state>`.

`list` adopts the standard error convention described in the [Errors](#errors) section.

## read
`read` is invoked as `<plugin_script> read <path> <state>`. When `read` is invoked, the script must output the entry's content.

`read` adopts the standard error convention described in the [Errors](#errors) section.

## metadata
`metadata` is invoked as `<plugin_script> metadata <path> <state>`. When `metadata` is invoked, the script must output a JSON object representing the entry's metadata. Below is an example of acceptable `metadata` output:

```json
{
  "key1": "value1",
  "key2": "value2"
}
```

`metadata` adopts the standard error convention described in the [Errors](#errors) section.

NOTE: Only implement `metadata` if there is additional information about your entry that is not provided by the `meta` attribute.

## stream
`stream` is invoked as `<plugin_script> stream <path> <state>`. When `stream` is invoked, the first line of the script's output must contain the `200` header. This header tells Wash that the entry's data is about to the streamed. After it outputs the header, the script must then stream the entry's data. Wash will continue to poll stdout for any updates until either the streaming process exits, or the user cancels the request.

`stream` adopts the standard error convention described in the [Errors](#errors) section.

## exec
`exec` is invoked as `<plugin_script> exec <path> <state> <opts> <cmd> <args...>`. If the `input` key is included as part of `opts` in a request to the `exec` endpoint, then its content is passed-in as stdin to the plugin script. `<opts>` is the JSON serialization of the remaining options.

When `exec` is invoked, the plugin script's stdout and stderr must be connected to `cmd`'s stdout and stderr, and it must exit the `exec` invocation with `cmd`'s exit code.

Because `exec` effectively hijacks `<plugin_script> exec` with `<cmd> <args...>`, there is currently no way for external plugins to report any `exec` errors to Wash. Thus, if `<plugin_script> exec` fails to exec `<cmd> <args...>` (e.g. due to a failed API call to trigger the exec), then that error output will be included as part of `<cmd> <args...>`'s output when running `wash exec`.

## Errors
All errors are printed to `stderr`. A method invocation is said to have errored when the plugin script returns a non-zero exit code. In that case, Wash wraps all of `stderr` into an error object, then documents that error in the process' activity and the server logs.

**NOTE:** Not all method invocations adopt this error handling convention (e.g. `exec`). The error handling for these "snowflake" methods is described in their respective sections.


## Aside (optional)
This section talks about the reasoning behind the plugin script's usage, shown below for convenience:

```s
<plugin_script> <method> <path> <state> <args...>
```

If we let `<entry> = <path> <state>`, then our usage becomes:

```s
<plugin_script> <method> <entry> <args...>
```

If we ignore `<plugin_script>` then the above turns into `<method> <entry> <args...>`. When read out loud, this looks like the function call `<method>(<entry>, <args...>)`. If we imagine `<entry>` as an object in an OOP language, this is semantically equivalent to:

```s
<entry>.<method>(<args...>)
```

For example, if `<entry> = myS3Bucket`, `<method> = list`, and `<args...>` is empty, then we can read the above as `myS3Bucket.list()`. Thus, the plugin script's usage can be thought of as invoking a method (`<method>`) on the specified entry (`<path> <state>`) with the given arguments (`<args...>`).

You might be wondering why we don't just lump `<path>` and `<state>` together into `<entry>` so that the plugin script's usage becomes `<plugin_script> <method> <entry> <args...>`. There's several reasons. One, having the `<path>` is helpful for debugging purposes. You can directly see the acting entry in the logs, which frees you from having to figure that information out yourself. Two, it mirrors the API's structure of `/fs/<method>/<path>`. And three, sometimes the `<path>` is all you need to write your plugin script. While you could always print the `<path>` yourself and make that the `<state>` parameter for Wash to pass around, it can be tedious to have to constantly do that, especially when you're writing simple plugins. Thus, `<path>` is really more of a convenience. You should use `<path>` if that's what you need to write your plugin. Otherwise, if you're writing a more complicated plugin that needs to maintain some state (e.g. like the entry's class name and its constructor arguments), then use `<state>`. However, try to avoid using `<path>` and `<state>` together in the same plugin script, as doing so can make it hard to reason about your code. Use either `<path>` or `<state>`, but not both.

**NOTE:** The `init` method is special. Its usage is `<plugin_script> init` -- there is no `<path>` or `<state`> so there is no `<entry>`. Thus, the OOP call of `<entry>.<method>(<args...>)` doesn't make sense for `init`. So how do you reason about it? Why do we have an `init` method? Since every Wash plugin is modeled as a filesystem, it must have a root. Once we know the root, then it is easy to get to a specific entry by repeatedly invoking the `list` method. The `init` method is how you describe that 'root'.

## Bash Example

[Download](./examples/sshfs.sh)

```s
{{< snippet "static/docs/external_plugins/examples/sshfs.sh" >}}
```
