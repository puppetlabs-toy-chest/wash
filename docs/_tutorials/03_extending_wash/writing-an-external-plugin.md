---
title: Writing an external plugin
---
In this tutorial, we'll write a plugin called `local_fs` that lets you navigate your local filesystem. Although the plugin isn’t practically useful,  this guide illustrates the key concepts behind external plugin development.

An external plugin consists of a *plugin* script. Wash shells out to this script whenever it needs to invoke an entry’s supported action (like *list*), or if it needs to query something about the entry (like its metadata). In a more general sense, Wash shells out to the plugin script whenever it needs to invoke an entry’s supported *method*. The invocation’s `stdout` typically contains the method’s result, while its `stderr` typically contains any errors. To list `local_fs`’ children, for example, Wash invokes `local_fs.sh list /local_fs`, then parses the appropriate entry objects from `stdout`.

**Note:** Technically, Wash would invoke `local_fs.sh list /local_fs ''`, where the third argument `''` (empty string) represents `local_fs`’ *state*. We are ignoring state in this tutorial, so the third argument will always be `''` for *all* plugin script invocations.

To get `local_fs` working, the first thing we need to do is implement the `init` method. Wash invokes `init` on startup to retrieve information about the external plugin’s root. Start by copying the following code into a `local_fs.sh` file:

{% highlight bash linenos %}
#!/bin/bash

set -e

function to_json_array {
  local list="$1"

  echo -n "["
  local has_multiple_elem=""
  for elem in ${list}; do
    if [[ -n ${has_multiple_elem} ]]; then
      echo -n ","
    else
      has_multiple_elem="true"
    fi
    echo -n "${elem}"
  done
  echo -n "]"
}

function print_entry_json() {
  local name="$1"
  local methods="$2"

  local attributes_json="$3"
  if [[ -z "${attributes_json}" ]]; then
    attributes_json="{}"
  fi

  local partial_metadata_json="$4"
  if [[ -z "${partial_metadata_json}" ]]; then
    partial_metadata_json="{}"
  fi

  local methods_json=`to_json_array "${methods}"`
  echo -n "{\
\"name\":\"${name}\",\
\"methods\":${methods_json},\
\"attributes\":${attributes_json},\
\"partial_metadata\": ${partial_metadata_json}\
}"
}

# This code implements 'init'
method="$1"
if [[ "${method}" == "init" ]]; then
  if [[ -z "${HOME}" ]]; then
    # Notice how we're printing errors to stderr then exiting with an
    # exit code of 1. The latter tells Wash that 'init' failed.
    echo 'The $HOME environment variable is not set.' 1>&2
    exit 1
  fi
  print_entry_json "local_fs" '"list"'
  echo ""
  exit 0
fi

echo "No other entries have been implemented." 1>&2
exit 1
{% endhighlight %}

**Note:** Don't forget to make `local_fs.sh` executable. `chmod +x /path/to/local_fs.sh` is one way of doing that.

Then invoke `local_fs init`. Your output should look something like this:

```
bash-3.2$ ./tutorials/local_fs.sh init
{"name":"local_fs","methods":["list"],"attributes":{},"partial_metadata":{}}
```

The printed JSON represents `local_fs`’ root. We see that the entry’s name is `local_fs`, that it implements `list`, and that it does not have any attributes or partial metadata.

Now that we’ve implemented `init`, let’s go ahead and see `local_fs` in action. Add the following to your `~/.puppetlabs/wash/wash.yaml` file:

```
external-plugins:
  - script: '/Users/enis.inan/GitHub/wash/tutorials/local_fs.sh'
```

Start-up Wash and enter an `ls`. You should see `local_fs` included in the output.

```
bash-3.2$ wash
Welcome to Wash!
  Wash includes several built-in commands: wexec, find, list, meta, tail.
  See commands run with wash via 'whistory', and logs with 'whistory <id>'.
Try 'help'
wash . ❯ ls
aws/
docker/
gcp/
kubernetes/
local_fs/
```

We're not done yet, so if you try to `ls local_fs`, you'll get this error:

```
wash . ❯ ls local_fs
puppetlabs.wash/errored-action: The list action errored on /Users/enis.inan/Library/Caches/wash/mnt863144881/local_fs: script returned a non-zero exit code of 1
COMMAND: (PID 74196) /Users/enis.inan/GitHub/wash/local_fs.sh list /local_fs ''
STDERR:
No other entries have been implemented.
```

Now let’s implement the rest of the `local_fs` plugin. Replace the code in `local_fs.sh` with:

{% highlight bash linenos %}
#!/bin/bash

set -e

function to_json_array {
  local list="$1"

  echo -n "["
  local has_multiple_elem=""
  for elem in ${list}; do
    if [[ -n ${has_multiple_elem} ]]; then
      echo -n ","
    else
      has_multiple_elem="true"
    fi
    echo -n "${elem}"
  done
  echo -n "]"
}

function print_entry_json() {
  local name="$1"
  local methods="$2"

  local attributes_json="$3"
  if [[ -z "${attributes_json}" ]]; then
    attributes_json="{}"
  fi

  local partial_metadata_json="$4"
  if [[ -z "${partial_metadata_json}" ]]; then
    partial_metadata_json="{}"
  fi

  local methods_json=`to_json_array "${methods}"`
  echo -n "{\
\"name\":\"${name}\",\
\"methods\":${methods_json},\
\"attributes\":${attributes_json},\
\"partial_metadata\": ${partial_metadata_json}\
}"
}

function bsd_stat_cmd() {
  local file="$1"
  stat -f '%a %m %c %Up %z %d %i %u %g' "${file}"
}

function gnu_stat_cmd() {
  local file="$1"
  stat_output=`stat -c '%X %Y %Z %f %s %d %i %u %g' "${file}"`
  local atime mtime ctime mode size device inode_number uid gid
  read atime mtime ctime mode size device inode_number uid gid <<< $stat_output
  mode=$((16#${mode}))

  echo $atime $mtime $ctime $mode $size $device $inode_number $uid $gid
}

function list_dir() {
  local dir="$1"
  # This code should output something like the following:
  #   [
  #     <child_json>,
  #     <child_json>,
  #     ...
  #     <child_json>
  #   ]
  echo "["
  local has_multiple_elem=""
  # Ignore hidden files to avoid weird shell-specific errors. If we don't
  # do this, then the script generates some weird error messages on ZSH.
  for file in `find "${dir}" -mindepth 1 -maxdepth 1 -not -path '*/.*'`; do
    # Make sure to add the trailing comma.
    if [[ -n ${has_multiple_elem} ]]; then
      echo ","
    else
      has_multiple_elem="true"
    fi

    # Get the file attributes using stat. Note that STAT_CMD is platform-specific
    # so be sure to change it to whatever's supported by your OS' stat command. The
    # subsequent comments provide some more guidance. In general, `STAT_CMD <file>`
    # should output the following info: 
    #   <atime> <mtime> <ctime> <mode> <size> <device> <inode_number> <uid> <gid>
    #
    # where the time attributes are in UNIX seconds, and all the other attributes
    # are decimal numbers.
    local STAT_CMD=""

    # Uncomment this if you're on OSX
    #local STAT_CMD="bsd_stat_cmd"

    # Uncomment this if you're on Linux
    #local STAT_CMD="gnu_stat_cmd"

    # Otherwise, you'll need to set STAT_CMD.
    #local STAT_CMD=""

    if [[ -z "${STAT_CMD}" ]]; then
      echo "Did not set STAT_CMD." 1>&2
      exit 1
    fi

    # Use stat to get the file's attributes.
    local stat_output=`${STAT_CMD} "${file}"`
    local atime mtime ctime mode size device inode_number uid gid
    read atime mtime ctime mode size device inode_number uid gid <<< $stat_output

    local methods
    if test -d "${file}"; then
      methods='"list"'
    else
      methods='"read"'
    fi

    local attributes_json="{\
\"atime\":${atime},\
\"mtime\":${mtime},\
\"ctime\":${ctime},\
\"size\":${size}\
}"

    local partial_metadata_json="{\
\"atime\":${atime},\
\"mtime\":${mtime},\
\"ctime\":${ctime},\
\"mode\":${mode},\
\"size\":${size},\
\"device\":${device},\
\"inodeNumber\":${inode_number},\
\"uid\":${uid},\
\"gid\":${gid}\
}"

    print_entry_json `basename "${file}"` "${methods}" "${attributes_json}" "${partial_metadata_json}"
  done
  echo ""
  echo -n "]"
}

# This code implements 'init'
method="$1"
if [[ "${method}" == "init" ]]; then
  if [[ -z "${HOME}" ]]; then
    # Notice how we're printing errors to stderr then exiting with an
    # exit code of 1. The latter tells Wash that 'init' failed.
    echo 'The $HOME environment variable is not set.' 1>&2
    exit 1
  fi
  print_entry_json "local_fs" '"list"'
  echo ""
  exit 0
fi

# This code implements all the other entries. Note that
# Wash only invokes supported methods. Thus, we don't
# have to worry about cases like reading a directory or
# listing a file's children.
path="$2"
path=`echo "${path}" | sed "s:^/local_fs:${HOME}:g"`
case "${method}" in
"list")
  list_dir "${path}"
  exit 0
;;
"read")
  cat "${path}"
  exit 0
;;
esac
{% endhighlight %}


**Note:** Don’t forget to set the `STAT_CMD` variable on line 88.

Now you can `ls local_fs`:

```
wash . ❯ ls local_fs/
Applications/
Desktop/
Documents/
Downloads/
GitHub/
Library/
Movies/
Music/
Pictures/
Public/
go/
```

Compare your output with `ls $HOME` (ignoring the hidden files).

Remember, when you use `ls local_fs`, which invokes the *list* action on the `local_fs` entry, Wash invokes `local_fs.sh list /local_fs` and parses its output. Let's see what happens when we invoke the script ourselves:

```
bash-3.2$ ./tutorials/local_fs.sh list /local_fs
[
{"name":"Music","methods":["list"],"attributes":{"atime":1576614404,"mtime":1575331025,"ctime":1575331025,"size":128},"partial_metadata": {"atime":1576614404,"mtime":1575331025,"ctime":1575331025,"mode":16832,"size":128,"device":16777221,"inodeNumber":12885648174,"uid":501,"gid":20}},
{"name":"go","methods":["list"],"attributes":{"atime":1576614404,"mtime":1576557876,"ctime":1576557876,"size":160},"partial_metadata": {"atime":1576614404,"mtime":1576557876,"ctime":1576557876,"mode":16877,"size":160,"device":16777221,"inodeNumber":12886708625,"uid":501,"gid":20}},
{"name":"Pictures","methods":["list"],"attributes":{"atime":1578617514,"mtime":1577074362,"ctime":1577074362,"size":128},"partial_metadata": {"atime":1578617514,"mtime":1577074362,"ctime":1577074362,"mode":16832,"size":128,"device":16777221,"inodeNumber":12885648177,"uid":501,"gid":20}},
{"name":"Desktop","methods":["list"],"attributes":{"atime":1578514087,"mtime":1577778220,"ctime":1577778220,"size":480},"partial_metadata": {"atime":1578514087,"mtime":1577778220,"ctime":1577778220,"mode":16832,"size":480,"device":16777221,"inodeNumber":12885648179,"uid":501,"gid":20}},
{"name":"Library","methods":["list"],"attributes":{"atime":1577838807,"mtime":1578958647,"ctime":1578958647,"size":1984},"partial_metadata": {"atime":1577838807,"mtime":1578958647,"ctime":1578958647,"mode":16832,"size":1984,"device":16777221,"inodeNumber":12885648155,"uid":501,"gid":20}},
{"name":"Public","methods":["list"],"attributes":{"atime":1575330350,"mtime":1575330349,"ctime":1575330350,"size":128},"partial_metadata": {"atime":1575330350,"mtime":1575330349,"ctime":1575330350,"mode":16877,"size":128,"device":16777221,"inodeNumber":12885648228,"uid":501,"gid":20}},
{"name":"GitHub","methods":["list"],"attributes":{"atime":1578973413,"mtime":1576614164,"ctime":1576614164,"size":512},"partial_metadata": {"atime":1578973413,"mtime":1576614164,"ctime":1576614164,"mode":16877,"size":512,"device":16777221,"inodeNumber":12886683114,"uid":501,"gid":20}},
{"name":"Movies","methods":["list"],"attributes":{"atime":1575330350,"mtime":1575330349,"ctime":1575330350,"size":96},"partial_metadata": {"atime":1575330350,"mtime":1575330349,"ctime":1575330350,"mode":16832,"size":96,"device":16777221,"inodeNumber":12885648232,"uid":501,"gid":20}},
{"name":"Applications","methods":["list"],"attributes":{"atime":1578514067,"mtime":1575331450,"ctime":1575331450,"size":96},"partial_metadata": {"atime":1578514067,"mtime":1575331450,"ctime":1575331450,"mode":16832,"size":96,"device":16777221,"inodeNumber":12885796512,"uid":501,"gid":20}},
{"name":"Documents","methods":["list"],"attributes":{"atime":1576522412,"mtime":1575325014,"ctime":1575337685,"size":1440},"partial_metadata": {"atime":1576522412,"mtime":1575325014,"ctime":1575337685,"mode":16832,"size":1440,"device":16777221,"inodeNumber":12885817316,"uid":501,"gid":20}},
{"name":"Downloads","methods":["list"],"attributes":{"atime":1578974744,"mtime":1578511137,"ctime":1578511137,"size":608},"partial_metadata": {"atime":1578974744,"mtime":1578511137,"ctime":1578511137,"mode":16832,"size":608,"device":16777221,"inodeNumber":12885648169,"uid":501,"gid":20}}
]
```

Each JSON object corresponds to a child entry. For example, the `Applications` directory’s JSON object is:

```
{
  "name": "Applications",
  "methods": ["list"],
  "attributes": {
    "atime": 1578514067,
    "mtime": 1575331450,
    "ctime": 1575331450,
    "size": 96
  },
  "partial_metadata": {
    "atime": 1578514067,
    "mtime": 1575331450,
    "ctime": 1575331450,
    "mode": 16832,
    "size": 96,
    "device": 16777221,
    "inodeNumber": 12885796512,
    "uid": 501,
    "gid": 20
  }
}
```

Notice that `list`'s output also included the children’s attributes. We can use `winfo` to check them out.

```
wash . ❯ winfo local_fs/Applications
Name: Applications
CName: Applications
Actions:
- list
Attributes:
  atime: 2020-01-08T12:07:47-08:00
  ctime: 2019-12-02T16:04:10-08:00
  mtime: 2019-12-02T16:04:10-08:00
  size: 96
```

**Note:** The `mode` attribute doesn't work on Mac OS so we are omitting it for now. However, we're still including it as metadata.

We can also use `meta` to check out each child’s metadata.

```
wash . ❯ meta local_fs/Applications
atime: 1578514067
ctime: 1575331450
device: 16777221
gid: 20
inodeNumber: 12885796512
mode: 16832
mtime: 1575331450
size: 96
uid: 501
```

Notice that the output matches the partial metadata. That’s because the partial metadata completely describes `local_fs`' children.

**Remember:** The partial metadata represents the raw response of a "bulk" fetch. For `local_fs`, the response would be `stat`’s output.

Congratulations! You've created your own Wash plugin to emulate some common file and directory filtering via the `local_fs` plugin and Wash `find`. The Wash external plugin interface, together with the attribute and metadata abstraction, make it possible for you to filter anything on almost anything! We can't wait to see what you do with Wash. Share your creations on the Wash Slack channel.

# Exercises
1. Try `cat`’ing a file in the `local_fs` plugin. What’s the underlying plugin script invocation? Hint: It’s similar to `list`.

    {% include exercise_answer.html answer="<code>local_fs.sh read &lt;path_to_file&gt;</code>" %}

2. Implement the `stream` action for files. Hint: Take a look at lines 109-114, and 155-170. Your implementation should be a wrapper to `tail -f`. Also, don’t forget to print the header!

    {% capture answer_2 %} 
    On line 113, add <code>stream</code> to the list of supported methods. Then add the following case to the <code>case</code> statement in line 155:
        <code><br />
          &nbsp;&nbsp;&nbsp;&nbsp;"stream")<br />
          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;echo "200"<br />
          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;tail -f "${path}"<br />
          &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;exit 0<br />
          &nbsp;&nbsp;&nbsp;&nbsp;;;<br />
        </code>
    {% endcapture %}
    {% include exercise_answer.html answer=answer_2 %}

3. How would you

    1. Find all `local_fs` entries that were modified within the last hour? Hint: `find local_fs -mtime -2h` gives you all `local_fs` entries that were modified within the last two hours.

        {% include exercise_answer.html answer="<code>find local_fs -mtime -1h</code>" %}

    2. Find all `local_fs` entries that are owned by the user with UID 10? Hint: `find local_fs -meta '.gid' 20` give you all `local_fs` entries with GID 20.

        {% include exercise_answer.html answer="<code>find local_fs -meta '.uid' 10</code>" %}

# Related Links
* [External plugin docs]({{ '/docs/external-plugins' | relative_url }})

# Next steps

That's the end of the _Extending Wash_ series! Click [here](../) to go back to the tutorials page.
