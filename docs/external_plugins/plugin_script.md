# Plugin Script
Wash shells out to the external plugin's script whenever it needs to invoke an action on one of its entries. The script must have the following usage:

```
<plugin_script> <action> <path> <state> <args...>
```

where

* `<action>` is the Wash action that's being invoked
* `<path>` is the entry's filesystem path relative to Wash's mountpoint
* `<state>` consists of the minimum amount of information required to reconstruct the entry inside the plugin
* `<args...>` are the arguments passed to the specific action.

`<path>` and `<state>` can be a bit confusing. To understand them, we recommend reading the [Aside](#aside), and to look at the provided [Bash](examples/sshfs.sh) + Ruby external plugin examples to see how they're used. **TODO: Link a Ruby example**

The remaining sections describe all possible values of `<action>` that can be passed-in, including each action's calling and error conventions, and the expected results.

## init
The `init` action is special. It is invoked as `<plugin_script> init`, and it is invoked only once, when the external plugin is loaded. When `init` is invoked, the script must output a JSON object representing the plugin root. Here's an example:

```
{
  "name": "aws",
  "supported_actions": [
    "list"
  ]
}
```

This example shows the *minimum* amount of information required for Wash to construct the plugin root. In fact, the example shows the minimum amount of information required for Wash to construct *any* external plugin entry. This information consists of the entry's name and its supported actions.

You can include additional keys in the printed JSON object. These keys are:

* `cache_ttls`. This specifies how many seconds each supported action's result should be cached (`ttl` is short for time to live).
* `attributes`. This represents the entry's filesystem attributes, if any. These are the access time `Atime`, last modified time `Mtime`, creation time `Ctime`, mode `Mode`, and size `Size`. The individual time attributes are specified in Unix seconds. Octal modes must be prefixed with the `0` delimiter (e.g. like `0777`). Hexadecimal modes must be prefixed with the `0x` delimiter (e.g. like `0xabcd`).
* `state`. This corresponds to the `<state>` parameter in the plugin script's usage.

Below is an example JSON object showcasing all possible keys at once.

```
{
  "name": "some_entry",
  "supported_actions": [
    "list"
  ],
  "cache_ttls": {
    "list": 30
  },
  "attributes": {
    "Atime": 1551942012,
    "Mtime": 1551942012,
    "Ctime": 1551942012,
    "Mode": 511,
    "Size": 15600
  },
  "state": "{\"klass\":\"AWS::Profile\"}"
}
```

We see from `cache_ttls` that the result of `some_entry`'s `list` action will be cached for 30 seconds. We see from `attributes` that `some_entry` has some filesystem attributes that are defined (note that the mode is `0777`; `511` is its Base-10 representation). Finally, we see from `state` that `some_entry` has some state that Wash will pass-back in via the `<state>` parameter whenever it invokes one of its supported actions. In this case, only `list` is supported, and `<state>` is a stringified JSON object containing the entry's class (`AWS::Profile`) in whatever language the plugin script was written in.

The `init` action adopts the standard error conventions described in the [Errors](#errors) section.

## list
The `list` action is invoked as `<plugin_script> list <path> <state>`. When `list` is invoked, the script must output an array of JSON objects. Each JSON object has the same schema as the JSON object described in the `init` section (hence, `list` outputs an array of entries). Below is an example of valid output from the `list` action.

```
[
	{
		"name": "LARGE_FILE.txt",
		"supported_actions": [
			"metadata",
			"read",
			"stream"
		],
		"attributes": {
			"Mtime": 1551459978,
			"Size": 100000000
		},
		"state": "{\"name\":\"LARGE_FILE.txt\",\"profile\":\"default\",\"region\":\"us-west-1\",\"bucket\":\"my-stupid-bucket-enis\",\"attributes\":{\"Mtime\":1551459978,\"Size\":100000000},\"klass\":\"S3Object\"}"
	},
	{
		"name": "my_json_file.json",
		"supported_actions": [
			"metadata",
			"read",
			"stream"
		],
		"attributes": {
			"Mtime": 1549572670,
			"Size": 9336
		},
		"state": "{\"name\":\"my_json_file.json\",\"profile\":\"default\",\"region\":\"us-west-1\",\"bucket\":\"my-stupid-bucket-enis\",\"attributes\":{\"Mtime\":1549572670,\"Size\":9336},\"klass\":\"S3Object\"}"
	}
]
```

**NOTE:** Remember that the state displayed here is the same `<state>` parameter that will be passed in to other actions invoked on that entry. For example, if the `read` action is invoked on `LARGE_FILE.txt` via `<plugin_script> read <parent_path>/LARGE_FILE.txt <state>`, then the value of `LARGE_FILE.txt`'s `state` key will be passed-in for `<state>`.

The `list` action adopts the standard error convention described in the [Errors](#errors) section.

## read
The `read` action is invoked as `<plugin_script> read <path> <state>`. When `read` is invoked, the script must output the entry's content.

The `read` action adopts the standard error convention described in the [Errors](#errors) section.

## metadata
The `metadata` action is invoked as `<plugin_script> metadata <path> <state>`. When `metadata` is invoked, the script must output a JSON object representing the entry's metadata. Below is an example of acceptable `metadata` output:

```
{
  "key1": "value1",
  "key2": "value2"
}
```

The `metadata` action adopts the standard error convention described in the [Errors](#errors) section.

## stream
The `stream` action is invoked as `<plugin_script> stream <path> <state>`. When `stream` is invoked, the first line of the script's output must contain the `200` header. This header tells Wash that the entry's data is about to the streamed. After it outputs the header, the script must then stream the entry's data. Wash will continue to poll stdout for any updates until either the streaming process exits, or the user cancels the request. In the latter case, Wash will send the `SIGTERM` signal to the streaming process.

The `stream` action adopts the standard error convention described in the [Errors](#errors) section.

## exec
The `exec` action is invoked as `<plugin_script> exec <path> <state> <cmd> <args...>`. If the `input` key is included as part of `opts` in a request to the `exec` endpoint, then its content is passed-in as stdin to the plugin script. When `exec` is invoked, the plugin script's stdout and stderr must be connected to `cmd`'s stdout and stderr, and it must exit the `exec` invocation with `cmd`'s exit code.

Because the `exec` action effectively hijacks `<plugin_script> exec` with `<cmd> <args...>`, there is currently no way for external plugins to report any `exec` errors to Wash. Thus, if `<plugin_script> exec` fails to exec `<cmd> <args...>` (e.g. due to a failed API call to trigger the exec), then that error output will be included as part of `<cmd> <args...>`'s output when running `wash exec`.

## Errors <a name="errors"></a>
All errors are printed to `stderr`. An action invocation is said to have errored when the plugin script returns a non-zero exit code. In that case, Wash wraps all of `stderr` into an error object, then documents that error in the process' journal and the server logs.

**NOTE:** Not all actions adopt this error handling convention (e.g. `exec`). The error handling for these "snowflake" actions is described in their respective sections.


## Aside (optional)<a name="aside"></a>
This section talks about the reasoning behind the plugin script's usage, shown below for convenience:

```
<plugin_script> <action> <path> <state> <args...>
```

If we let `<entry> = <path> <state>`, then our usage becomes:

```
<plugin_script> <action> <entry> <args...>
```

If we ignore `<plugin_script>` then the above turns into `<action> <entry> <args...>`. When read out loud, this looks like the function call `<action>(<entry>, <args...>)`. If we imagine `<entry>` as an object in an OOP language, this is semantically equivalent to:

```
<entry>.<action>(<args...>)
```

For example, if `<entry> = myS3Bucket`, `<action> = list`, and `<args...>` is empty, then we can read the above as `myS3Bucket.list()`. Thus, the plugin script's usage can be thought of as invoking an action (`<action>`) on the specified entry (`<path> <state>`) with the given arguments (`<args...>`).

You might be wondering why we don't just lump `<path>` and `<state>` together into `<entry>` so that the plugin script's usage becomes `<plugin_script> <action> <entry> <args...>`. There's several reasons. One, having the `<path>` is helpful for debugging purposes. You can directly see the acting entry in the logs, which frees you from having to figure that information out yourself. Two, it mirrors the API's structure of `/fs/<action>/<path>`. And three, sometimes the `<path>` is all you need to write your plugin script. While you could always print the `<path>` yourself and make that the `<state>` parameter for Wash to pass around, it can be annoying to have to constantly do that, especially when you're writing simple plugins. Thus, `<path>` is really more of a convenience. You should use `<path>` if that's what you need to write your plugin. Otherwise, if you're writing a more complicated plugin that needs to maintain some state (e.g. like the entry's class name and its constructor arguments), then use `<state>`. However, try to avoid using `<path>` and `<state>` together in the same plugin script, as doing so can make it hard to reason about your code. Use either `<path>` or `<state>`, but not both.

**NOTE:** The `init` action is special. Its usage is `<plugin_script> init` -- there is no `<path>` or `<state`> so there is no `<entry>`. Thus, the OOP call of `<entry>.<action>(<args...>)` doesn't make sense for `init`. So how do you reason about it? Why do we have an `init` action? Since every Wash plugin is modeled as a filesystem, it must have a root. Once we know the root, then it is easy to get to a specific entry by repeatedly invoking the `list` action. The `init` action is how you describe that 'root'.