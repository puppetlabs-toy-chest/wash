---
title: Writing an external plugin
---
In this tutorial, we will write the `local_fs` plugin. `local_fs` lets you navigate your local filesystem. Although it isn’t practically useful, it illustrates the key concepts behind external plugin development.

An external plugin consists of a plugin script. Wash shells out to this script whenever it needs to invoke an entry’s supported action (like `list`), or if it needs to query something about the entry (like its metadata). In a more general sense, Wash shells out to the plugin script whenever it needs to invoke an entry’s supported method. The invocation’s `stdout` will (typically) contain the method’s result, while its `stderr` will (typically) contain any errors. To list `local_fs`' children, for example, Wash will invoke `local_fs.sh list /local_fs`[^1], then parse the appropriate entry objects from stdout.

[^1]: Technically, Wash would invoke `local_fs.sh list /local_fs ''`, where the third argument `''` (empty string) represents `local_fs`’ state. We are ignoring state in this tutorial, so the third argument will always be `''` for all plugin script invocations.

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
  local methods_json=`to_json_array "${methods}"`
  echo -n "{\
\"name\":\"${name}\",\
\"methods\":${methods_json},\
\"attributes\":${attributes_json}\
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

Then invoke `local_fs init`. Your output should look something like

```
bash-3.2$ ./tutorials/local_fs.sh init
{"name":"local_fs","methods":["list"],"attributes":{}}
```

The printed JSON represents `local_fs`’ root. We see that the entry’s name is `local_fs`, that it implements `list`, and that it does not have any attributes. In fact, we will soon see that every entry will be represented by similar JSON. You can also deduce this fact from the `print_entry_json` helper’s name on line 46.

Now that we’ve implemented `init`, let’s go ahead and see `local_fs` in action. Add the following to your `~/.puppetlabs/wash/wash.yaml` file:

```
external-plugins:
  - script: '/Users/enis.inan/GitHub/wash/tutorials/local_fs.sh'
```

and start-up Wash. You should see `local_fs` included in `ls`' output.

```
bash-3.2$ wash
Welcome to Wash!
  Wash includes several built-in commands: wexec, find, list, meta, tail.
  See commands run with wash via 'whistory', and logs with 'whistory <id>'.
Try 'help'
wash . ❯ ls
aws        docker     gcp        kubernetes local_fs
```

If you try to `ls local_fs`, you should get an error

```
wash . ❯ ls local_fs
WARN FUSE: List /local_fs errored: script returned a non-zero exit code of 1
COMMAND: (PID 60927) /Users/enis.inan/GitHub/wash/tutorials/local_fs.sh list /local_fs ''
STDERR:
No other entries have been implemented.
WARN FUSE: List /local_fs errored: script returned a non-zero exit code of 1
COMMAND: (PID 60927) /Users/enis.inan/GitHub/wash/tutorials/local_fs.sh list /local_fs ''
STDERR:
No other entries have been implemented.
WARN FUSE: List /local_fs errored: script returned a non-zero exit code of 1
COMMAND: (PID 60927) /Users/enis.inan/GitHub/wash/tutorials/local_fs.sh list /local_fs ''
STDERR:
No other entries have been implemented.
WARN FUSE: List /local_fs errored: script returned a non-zero exit code of 1
COMMAND: (PID 60927) /Users/enis.inan/GitHub/wash/tutorials/local_fs.sh list /local_fs ''
STDERR:
No other entries have been implemented.
WARN FUSE: List /local_fs errored: script returned a non-zero exit code of 1
COMMAND: (PID 60927) /Users/enis.inan/GitHub/wash/tutorials/local_fs.sh list /local_fs ''
STDERR:
No other entries have been implemented.
WARN FUSE: List /local_fs errored: script returned a non-zero exit code of 1
COMMAND: (PID 60927) /Users/enis.inan/GitHub/wash/tutorials/local_fs.sh list /local_fs ''
STDERR:
No other entries have been implemented.
WARN FUSE: List /local_fs errored: script returned a non-zero exit code of 1
COMMAND: (PID 60927) /Users/enis.inan/GitHub/wash/tutorials/local_fs.sh list /local_fs ''
STDERR:
No other entries have been implemented.
WARN FUSE: List /local_fs errored: script returned a non-zero exit code of 1
COMMAND: (PID 60927) /Users/enis.inan/GitHub/wash/tutorials/local_fs.sh list /local_fs ''
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
  local methods_json=`to_json_array "${methods}"`
  echo -n "{\
\"name\":\"${name}\",\
\"methods\":${methods_json},\
\"attributes\":${attributes_json}\
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
\"size\":${size},\
\"meta\":{\
  \"atime\":${atime},\
  \"mtime\":${mtime},\
  \"ctime\":${ctime},\
  \"mode\":${mode},\
  \"size\":${size},\
  \"device\":${device},\
  \"inodeNumber\":${inode_number},\
  \"uid\":${uid},\
  \"gid\":${gid}\
}}"

    print_entry_json `basename "${file}"` "${methods}" "${attributes_json}"
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


**Note:** Don’t forget to set the `STAT_CMD` variable on line 81.

`ls`’ing local_fs should work now:

```
wash . ❯ ls local_fs
Applications      GitHub            PGitHub           Tutorials         svn
Desktop           Installs          Pictures          Upload            workspace
Documents         Library           Public            UsefulScripts
Downloads         Misc              Ruby              eclipse-workspace
Dump              Movies            Sites             go
Experiments       Music             TranslationRepos  support-tool
```

(You should compare your output with `ls $HOME`).

Recall that `ls local_fs` invokes the `list` action on the `local_fs` entry, which reduced to invoking `local_fs.sh list /local_fs` and parsing its output. Let’s check out the latter.

```
bash-3.2$ ./tutorials/local_fs.sh list /local_fs
[
{"name":"svn","methods":["list"],"attributes":{"atime":1569968997,"mtime":1518042412,"ctime":1532474427,"size":128,"meta":{  "atime":1569968997,  "mtime":1518042412,  "ctime":1532474427,  "mode":16877,  "size":128,  "device":16777220,  "inodeNumber":2425576,  "uid":503,  "gid":20}}},
{"name":"Misc","methods":["list"],"attributes":{"atime":1532474928,"mtime":1501020816,"ctime":1532474222,"size":128,"meta":{  "atime":1532474928,  "mtime":1501020816,  "ctime":1532474222,  "mode":16877,  "size":128,  "device":16777220,  "inodeNumber":2408689,  "uid":503,  "gid":20}}},
{"name":"Music","methods":["list"],"attributes":{"atime":1532475166,"mtime":1516903352,"ctime":1532474458,"size":128,"meta":{  "atime":1532475166,  "mtime":1516903352,  "ctime":1532474458,  "mode":16832,  "size":128,  "device":16777220,  "inodeNumber":2408694,  "uid":503,  "gid":20}}},
{"name":"Dump","methods":["list"],"attributes":{"atime":1532474928,"mtime":1506572473,"ctime":1532471948,"size":192,"meta":{  "atime":1532474928,  "mtime":1506572473,  "ctime":1532471948,  "mode":16877,  "size":192,  "device":16777220,  "inodeNumber":978092,  "uid":503,  "gid":20}}},
{"name":"go","methods":["list"],"attributes":{"atime":1549159270,"mtime":1549423958,"ctime":1549423958,"size":160,"meta":{  "atime":1549159270,  "mtime":1549423958,  "ctime":1549423958,  "mode":16877,  "size":160,  "device":16777220,  "inodeNumber":13767491,  "uid":503,  "gid":20}}},
{"name":"Installs","methods":["list"],"attributes":{"atime":1532475166,"mtime":1498666826,"ctime":1532474104,"size":128,"meta":{  "atime":1532475166,  "mtime":1498666826,  "ctime":1532474104,  "mode":16877,  "size":128,  "device":16777220,  "inodeNumber":2365626,  "uid":503,  "gid":20}}},
{"name":"TranslationRepos","methods":["list"],"attributes":{"atime":1532474931,"mtime":1502748289,"ctime":1532474427,"size":64,"meta":{  "atime":1532474931,  "mtime":1502748289,  "ctime":1532474427,  "mode":16877,  "size":64,  "device":16777220,  "inodeNumber":2588564,  "uid":503,  "gid":20}}},
{"name":"Experiments","methods":["list"],"attributes":{"atime":1560967858,"mtime":1560967858,"ctime":1560967858,"size":96,"meta":{  "atime":1560967858,  "mtime":1560967858,  "ctime":1560967858,  "mode":16877,  "size":96,  "device":16777220,  "inodeNumber":31253883,  "uid":503,  "gid":20}}},
{"name":"Pictures","methods":["list"],"attributes":{"atime":1532475166,"mtime":1517959296,"ctime":1532474458,"size":160,"meta":{  "atime":1532475166,  "mtime":1517959296,  "ctime":1532474458,  "mode":16832,  "size":160,  "device":16777220,  "inodeNumber":2425431,  "uid":503,  "gid":20}}},
{"name":"workspace","methods":["list"],"attributes":{"atime":1551289602,"mtime":1551289602,"ctime":1551289602,"size":96,"meta":{  "atime":1551289602,  "mtime":1551289602,  "ctime":1551289602,  "mode":16877,  "size":96,  "device":16777220,  "inodeNumber":15771590,  "uid":503,  "gid":20}}},
{"name":"Desktop","methods":["list"],"attributes":{"atime":1569991296,"mtime":1569902573,"ctime":1569902573,"size":1472,"meta":{  "atime":1569991296,  "mtime":1569902573,  "ctime":1569902573,  "mode":16832,  "size":1472,  "device":16777220,  "inodeNumber":944178,  "uid":503,  "gid":20}}},
{"name":"Library","methods":["list"],"attributes":{"atime":1570041461,"mtime":1569360574,"ctime":1569360574,"size":2240,"meta":{  "atime":1570041461,  "mtime":1569360574,  "ctime":1569360574,  "mode":16832,  "size":2240,  "device":16777220,  "inodeNumber":2365902,  "uid":503,  "gid":20}}},
{"name":"eclipse-workspace","methods":["list"],"attributes":{"atime":1532475163,"mtime":1499457703,"ctime":1532471948,"size":96,"meta":{  "atime":1532475163,  "mtime":1499457703,  "ctime":1532471948,  "mode":16877,  "size":96,  "device":16777220,  "inodeNumber":978097,  "uid":503,  "gid":20}}},
{"name":"Sites","methods":["list"],"attributes":{"atime":1532474938,"mtime":1532474458,"ctime":1532474458,"size":96,"meta":{  "atime":1532474938,  "mtime":1532474458,  "ctime":1532474458,  "mode":16877,  "size":96,  "device":16777220,  "inodeNumber":2618702,  "uid":503,  "gid":20}}},
{"name":"Public","methods":["list"],"attributes":{"atime":1532475089,"mtime":1495831550,"ctime":1532474255,"size":128,"meta":{  "atime":1532475089,  "mtime":1495831550,  "ctime":1532474255,  "mode":16877,  "size":128,  "device":16777220,  "inodeNumber":2425566,  "uid":503,  "gid":20}}},
{"name":"GitHub","methods":["list"],"attributes":{"atime":1570031837,"mtime":1569639580,"ctime":1569639580,"size":8064,"meta":{  "atime":1570031837,  "mtime":1569639580,  "ctime":1569639580,  "mode":16877,  "size":8064,  "device":16777220,  "inodeNumber":978173,  "uid":503,  "gid":20}}},
{"name":"Movies","methods":["list"],"attributes":{"atime":1532475144,"mtime":1495831550,"ctime":1532474458,"size":96,"meta":{  "atime":1532475144,  "mtime":1495831550,  "ctime":1532474458,  "mode":16832,  "size":96,  "device":16777220,  "inodeNumber":2408692,  "uid":503,  "gid":20}}},
{"name":"Applications","methods":["list"],"attributes":{"atime":1570031244,"mtime":1495831999,"ctime":1532471853,"size":128,"meta":{  "atime":1570031244,  "mtime":1495831999,  "ctime":1532471853,  "mode":17344,  "size":128,  "device":16777220,  "inodeNumber":943924,  "uid":503,  "gid":20}}},
{"name":"Tutorials","methods":["list"],"attributes":{"atime":1569641420,"mtime":1569641334,"ctime":1569641334,"size":704,"meta":{  "atime":1569641420,  "mtime":1569641334,  "ctime":1569641334,  "mode":16877,  "size":704,  "device":16777220,  "inodeNumber":2588565,  "uid":503,  "gid":20}}},
{"name":"Documents","methods":["list"],"attributes":{"atime":1565821490,"mtime":1565821494,"ctime":1565821494,"size":1440,"meta":{  "atime":1565821490,  "mtime":1565821494,  "ctime":1565821494,  "mode":16832,  "size":1440,  "device":16777220,  "inodeNumber":944192,  "uid":503,  "gid":20}}},
{"name":"support-tool","methods":["list"],"attributes":{"atime":1532475144,"mtime":1501172807,"ctime":1532474255,"size":64,"meta":{  "atime":1532475144,  "mtime":1501172807,  "ctime":1532474255,  "mode":16877,  "size":64,  "device":16777220,  "inodeNumber":2425575,  "uid":503,  "gid":20}}},
{"name":"UsefulScripts","methods":["list"],"attributes":{"atime":1532475143,"mtime":1508201220,"ctime":1532474458,"size":160,"meta":{  "atime":1532475143,  "mtime":1508201220,  "ctime":1532474458,  "mode":16877,  "size":160,  "device":16777220,  "inodeNumber":2618684,  "uid":503,  "gid":20}}},
{"name":"Downloads","methods":["list"],"attributes":{"atime":1569859792,"mtime":1569859792,"ctime":1569859792,"size":1024,"meta":{  "atime":1569859792,  "mtime":1569859792,  "ctime":1569859792,  "mode":16832,  "size":1024,  "device":16777220,  "inodeNumber":975831,  "uid":503,  "gid":20}}},
{"name":"Ruby","methods":["list"],"attributes":{"atime":1532475143,"mtime":1507151974,"ctime":1532474255,"size":64,"meta":{  "atime":1532475143,  "mtime":1507151974,  "ctime":1532474255,  "mode":16877,  "size":64,  "device":16777220,  "inodeNumber":2425574,  "uid":503,  "gid":20}}},
{"name":"Upload","methods":["list"],"attributes":{"atime":1532475143,"mtime":1507180839,"ctime":1532474458,"size":128,"meta":{  "atime":1532475143,  "mtime":1507180839,  "ctime":1532474458,  "mode":16877,  "size":128,  "device":16777220,  "inodeNumber":2618681,  "uid":503,  "gid":20}}},
{"name":"PGitHub","methods":["list"],"attributes":{"atime":1569859882,"mtime":1552437895,"ctime":1552437895,"size":480,"meta":{  "atime":1569859882,  "mtime":1552437895,  "ctime":1552437895,  "mode":16877,  "size":480,  "device":16777220,  "inodeNumber":2408772,  "uid":503,  "gid":20}}}
]
```

Each JSON object corresponds to a child entry. For example, the `Applications` directory’s JSON object is 

```
{
  "name": "Applications",
  "methods": [
    "list"
  ],
  "attributes": {
    "atime": 1570031244,
    "mtime": 1495831999,
    "ctime": 1532471853,
    "size": 128,
    "meta": {
      "atime": 1570031244,
      "mtime": 1495831999,
      "ctime": 1532471853,
      "mode": 17344,
      "size": 128,
      "device": 16777220,
      "inodeNumber": 943924,
      "uid": 503,
      "gid": 20
    }
  }
}
```

Notice that `list`'s output also included the children’s attributes[^2]. We can use `winfo` to check them out.

[^2]: The `mode` attribute doesn't work on Mac OS so we are omitting it for now. However, we're still including it as metadata.

```
wash . ❯ winfo local_fs/Applications
Path: /Users/enis.inan/Library/Caches/wash/mnt182709991/local_fs/Applications
Name: Applications
CName: Applications
Actions:
- list
Attributes:
  atime: 2019-10-02T08:47:24-07:00
  ctime: 2018-07-24T15:37:33-07:00
  mtime: 2017-05-26T13:53:19-07:00
  size: 128
```

We can also use `meta` to check out each child’s metadata.

```
wash . ❯ meta local_fs/Applications
atime: 1570031244
ctime: 1532471853
device: 16777220
gid: 20
inodeNumber: 943924
mode: 17344
mtime: 1495831999
size: 128
uid: 503
```

Notice that the output matches the `meta` attribute[^3]. That’s because the `meta` attribute completely describes `local_fs`' children.

[^3]: Recall that the `meta` attribute represented the raw response of a "bulk" fetch. For `local_fs`, the response would be `stat`’s output.

# Exercises
1. Try `cat`’ing a file in the `local_fs` plugin. What’s the underlying plugin script invocation? Hint: It’s similar to `list`.

    {% include exercise_answer.html answer="<code>local_fs.sh read &lt;path_to_file&gt;</code>" %}

2. Implement the `stream` action for files. Hint: Take a look at lines 102-107, and 146-161. Your implementation should be a wrapper to `tail -f`. Also, don’t forget to print the header!

    {% capture answer_2 %} 
    On line 106, add <code>stream</code> to the list of supported methods. Then add the following case to the <code>case</code> statement in line 152:
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

   The purpose of this exercise was to show you that Wash’s external plugin interface and attributes and metadata abstraction make it possible for you to filter anything on almost anything. Here, we were able to emulate some common file and directory filtering via the `local_fs` plugin and `find`. You can learn more about `find` in the [Filtering entries with find](../02_find) tutorial.

# Related Links
* [External plugin docs]({{ '/docs/external-plugins' | relative_url }})

# Next steps

That's the end of the _Extending Wash_ series! Click [here](../) to go back to the tutorials page.
