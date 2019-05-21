#!/usr/bin/env bash

# This examples implements an SSH filesystem in Bash.
# The filesystem is laid out as:
#   /sshfs
#     <vm1>
#       <vm1_root_dir>
#     <vm2>
#       <vm2_root_dir>
#     ...
#
# For simplicity, our SSH filesystem consists of two VMs
# specified by the SSHFS_VM_ONE and SSHFS_VM_TWO environment
# variables. The VMs are listed by their fqdns. It is assumed
# that both VMs are known hosts, and that you can ssh into them
# as the root user. For example, it assumes that something like
# "ssh root@<vm> ls" works.
#
# If you'd like to try this plugin out, then add the following to
# your plugins.yaml file:
#
#   - name: 'sshfs'
#     script: '<wash_parent>/wash/docs/external_plugins/examples/sshfs.sh'
#
# and start-up the Wash server

# Below are some helpers. The main script comes after them.

# Gets the root directory of a given path. For example, if
# path is something like /sshfs/<vm>/..., then this returns
# /sshfs
function get_root {
  local path="$1"

  echo "${path}" | sed -E 's:.*^/([^/]+).*:\1:'
}

# Strips the root directory of a given path. For example, if
# path is something like /sshfs/<vm>/..., then this returns
# /<vm>/...
function strip_root {
  local path="$1"

  echo "${path}" | sed -E 's:^/([^/]+)::'
}

function vm_exec {
  local vm="$1"
  local cmd="$2"

  local tty="$3"
  if [[ -z "${tty}" || "${tty}" == "true" ]]; then
    # Wash can prematurely kill our process while the remote SSH
    # command is running. Setting up a tty ensures that the remote
    # SSH command is killed when the calling process (our
    # process) is killed. This avoids orphaned SSH processes.
    #
    # NOTE: We make TTY optional because it is one of the passed-in
    # Exec options.
    #
    # NOTE: This type of plugin-specific cleanup is the plugin
    # author's responsibility, not Wash's.
    ssh -tt root@"${vm}" "${cmd}"
  else
    ssh root@"${vm}" "${cmd}"
  fi
}

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

# Prints the given entry's JSON object. This is used by `init`
# and `list`.
#
# Remember, only the entry name and its implemented methods
# are required. The attributes should be provided if your
# entry's a resource (e.g. like a file, container, VM, database,
# etc.).
function print_entry_json() {
  local name="$1"
  local methods="$2"

  local attributes_json="$3"
  if [[ -z "${attributes_json}" ]]; then
    # The attributes_json is optional. We chose to print something
    # here to make the code a little cleaner. Don't worry, Wash knows
    # that an empty attributes JSON means that the entry doesn't have
    # any attributes.
    attributes_json="{}"
  fi
  local methods_json=`to_json_array "${methods}"`
  echo "{\
\"name\":\"${name}\",\
\"methods\":${methods_json},\
\"attributes\":${attributes_json}\
}"
}

# Prints the given file's JSON object. The file parameters are parsed from
# its corresponding stat output.
function print_file_json {
  local isDir="$1"
  local size="$2"
  local atime="$3"
  local mtime="$4"
  local ctime="$5"
  local mode="$6"
  local path="$7"

  name=`basename "${path}"`
  intMode=$((16#${mode}))
  if [[ ${isDir} -eq 0 ]]; then
    methods='"list"'
    # Unfortunately, Wash doesn't handle symlinks well. Thus
    # for now, we'll assume that sym-linked directories are
    # regular directories.
    intMode=$((${intMode} | 16384))
  else
    methods='"read" "stream"'
  fi
  # We could include additional information about the
  # file/directory in the "meta" attribute (e.g. like its
  # inode number), but doing so complicates the code a
  # bit.
  attributes_json="{\
\"atime\":${atime},\
\"mtime\":${mtime},\
\"ctime\":${ctime},\
\"mode\":${intMode},\
\"size\":${size}\
}"
  print_entry_json "${name}" "${methods}" "${attributes_json}"
}

# Prints the children of the specified directory. The code here
# does a few optimizations with xargs to make this sshfs plugin
# fast-enough for interactive use. However, the key-takeaway here
# is the end-result: that stdout contains an array of JSON objects
# corresponding to the directory's children.
function print_children {
  local vm="$1"
  local dir="$2"

  # The code here is equivalent to ls'ing the directory, then invoking
  # `test-d` and stat on each entry to obtain the following information:
  #   <is_dir> <sizeAttr> <atime> <mtime> <ctime> <mode> <path>
  #
  # TODO: Handle TTY strangeness in the returned output. For now, it is
  # enough to set it to false.
  stat_output=`vm_exec ${vm} "find ${dir} -mindepth 1 -maxdepth 1 -exec bash -c 'test -d \\$0; echo -n \"\\$? \"' {} \; -exec stat -c '%s %X %Y %Z %f %n' {} \;" false`
  if [[ -z "${stat_output}" ]]; then
    echo "[]"
    return 0
  fi

  # Each line of stat_output is a child. The following code creates a pipeline
  # that takes each line, converts it to the corresponding child JSON object, then
  # prints the result. We stream the JSON objects for performance reasons. The important
  # bit here is to note that after print_children exits, stdout should look something like:
  #
  # [
  # <child_json>,
  # <child_json>,
  # ...
  # <child_json> 
  # ]
  #
  # which satisfies Wash.
  echo "["
  # Print all children except for the last child.
  num_children=`echo "${stat_output}" | wc -l | awk '{print $1}'`
  if [[ num_children -gt 1 ]]; then
    export -f to_json_array
    export -f print_file_json
    export -f print_entry_json

    echo "${stat_output}"\
       | head -n$((num_children-1))\
       | xargs -n7 -P 10 -I {} bash -c 'print_file_json $@' _ {}\
       | sed s/$/,/
  fi
  # Now print the last child
  print_file_json `echo "${stat_output}" | tail -n1`
  echo "]"
}

method="$1"
if [[ "${method}" == "init" ]]; then
  # Our root's name is "sshfs." It only implements "list"
  print_entry_json "sshfs" '"list"'
  exit 0
fi

path="$2"

path=`strip_root ${path}`
if [[ "${path}" == "" ]]; then
  # Wash is invoking a method on our root. Since Wash only passes
  # in implemented methods, and since our root only implements
  # "list", we can assume that method == "list" here.
  #
  # Since we've structured our filesystem as /sshfs/<vm>/...,
  # "listing" our root consists of listing the VMs.
  function print_vm_json() {
      local name="$1"
      local methods='"list" "exec" "metadata"'
      # A VM can be modified, so some sort of mtime attribute
      # makes sense. The other attributes (ctime, atime, mode,
      # size) don't make sense, so don't set them. Notice how we
      # also include the VM's partial metadata via the "meta" attribute.
      # This would typically be the raw JSON object returned by
      # an API's "list" endpoint (e.g. like a "/vms" REST endpoint).
      # However, since we're not using any kind of API in our sshfs
      # example, we'll just set "meta" to something random.
      #
      # NOTE: The mtime is in Unix seconds. It corresponds to
      # May 17th, 2019 at 3:15 AM UTC. We recommend passing back
      # Unix seconds for all your time-attribute values since they
      # are the easiest for Wash to parse.
      local mtime="1558062927"
      local attributes="{\
\"mtime\":${mtime},\
\"meta\":{\
\"LastModifiedTime\":${mtime},\
\"Owner\":\"Alice\"\
}\
}"

      print_entry_json "${name}" "${methods}" "${attributes}"
  }

  to_json_array "`print_vm_json ${SSHFS_VM_ONE}` `print_vm_json ${SSHFS_VM_TWO}`"
  exit 0
fi

# path is of the form /<vm>/... so get the VM's name
vm=`get_root ${path}`

path=`strip_root ${path}`
if [[ "${path}" == "" ]]; then
  # The method's being invoked on a VM. Since a VM implements "list",
  # "exec", and "metadata", we case our code on those methods.
  case "${method}" in
  "list")
    # "list"'ing a VM is equivalent to listing its root
    print_children ${vm} "/"
    exit 0
  ;;
  "exec")
    opts="$4"
    cmd="$5"
    shift
    shift
    shift
    shift
    shift
    args="$@"

    # First we parse the provided Exec options. Wash guarantees that
    # the passed-in options are valid JSON and that they are all present,
    # so we don't need to do our own validation.
    #
    # NOTE: Only the "tty" option is relevant. We don't need to worry about
    # "elevate" because we are already running our commands as root.
    tty=`echo "${opts}" | jq .tty`

    # Now we exec the command and exit with its exit code.
    #
    # NOTE: Our process' Stdin is the content of the "Stdin" Exec option.
    cat /dev/stdin | vm_exec "${vm}" "${cmd} ${args}" "${tty}"
    exit "$?"
  ;;
  "metadata")
    # Wash is requesting the VM's full metadata.
    #
    # NOTE: Only implement "metadata" if there is additional information
    # about your resource that is not provided by the "meta" attribute.
    # In our example, the additional information is the VM's platform.
    #
    # NOTE: Since "metadata" is meant to return a complete description of
    # the entry, it should be a superset of the "meta" attribute.
    echo "{\
\"LastModifiedTime\":1558062927,\
\"Owner\":\"Alice\",\
\"Platform\":\"CentOS\"\
}"
    exit 0
  ;;
  *)
    # We print errors to stderr then exit with a non-zero
    # exit code. This tells Wash that our invocation failed.
    echo "missing a case statement for the '${method}'' method" >2
    exit 1
  ;;
  esac
fi

# Our path is an absolute path in the VM's filesystem.
# Thus, we can just case on all the possible methods that
# can be passed-in.
case "${method}" in
"list")
  print_children ${vm} "${path}/"
  exit 0
;;
"read")
  vm_exec "${vm}" "cat ${path}"
  exit 0
;;
"stream")
  # Notice how we print the "200" header first before anything
  # else. In HTTP, "200" is the "OK" status code. Thus, printing
  # this header tells Wash that everything's "OK", and that we
  # are about to stream some stuff.
  echo "200"

  # Use `tail -f` to stream the file's content. Wash will send
  # the SIGTERM signal to our process when it no longer needs
  # our streamed data.
  vm_exec "${vm}" "tail -f ${path}"
  exit 0
;;
*)
  echo "missing a case statement for the '${method}' method" >2
  exit 1
;;
esac