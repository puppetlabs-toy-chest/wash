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

  ssh root@"${vm}" "${cmd}"
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

# Prints the given entry's JSON object. This is used by the
# `init` and `list` actions.
#
# Remember, only the entry name and supported actions are
# required.
function print_entry_json() {
  local name="$1"
  local supported_actions="$2"

  local attributes_json="$3"
  if [[ -z "${attributes_json}" ]]; then
    # The attributes_json is optional. We chose to print something
    # here to make the code a little cleaner. Don't worry, Wash knows
    # that an empty attributes object translates to the entry not having
    # any filesystem attributes.
    attributes_json="{}"
  fi

  local supported_actions_json=`to_json_array "${supported_actions}"`

  echo "{\
\"name\":\"${name}\",\
\"supported_actions\":${supported_actions_json},\
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

  intMode=$(echo $((16#${mode})))
  if [[ ${isDir} -eq 0 ]]; then
    supported_actions='"list"'

    # Unfortunately, Wash doesn't handle symlinks well. Thus
    # for now, we'll assume that sym-linked directories are
    # regular directories.
    intMode=$(echo $((${intMode} | 16384)))
  else
    supported_actions='"read" "stream"'
  fi

  attributes_json=$(echo "{\
\"Atime\":${atime},\
\"Mtime\":${mtime},\
\"Ctime\":${ctime},\
\"Mode\":${intMode},\
\"Size\":${size}\
}")

  print_entry_json "${name}" "${supported_actions}" "${attributes_json}"
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
  stat_output=`vm_exec ${vm} "find ${dir} -mindepth 1 -maxdepth 1 | xargs -n1 -r -I {} bash -c '(test -d \\$@; echo -n \"\\$? \") && stat -c \"%s %X %Y %Z %f %n\" \\$@' _ {}"`
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
    export -f vm_exec
    
    echo "${stat_output}"\
       | head -n$((num_children-1))\
       | xargs -n7 -P 10 -I {} bash -c 'print_file_json $@' _ {}\
       | sed s/$/,/
  fi

  # Now print the last child
  print_file_json `echo "${stat_output}" | tail -n1`

  echo "]"
}

action="$1"
if [[ "${action}" == "init" ]]; then
  # Our root's name is "sshfs." It only supports the "list"
  # action.
  print_entry_json "sshfs" '"list"'
  exit 0
fi

path="$2"

path=`strip_root ${path}`
if [[ "${path}" == "" ]]; then
  # Our action's being invoked on the root. Since Wash only passes
  # in supported actions, and since our root only supports the
  # "list" action, we can assume that action == "list" here.
  #
  # Since we've structured our filesystem as /sshfs/<vm>/...,
  # ls'ing our root consists of listing the VMs.
  function print_vm_json() {
      local name="$1"

      # VMs support the "list", "exec" and "metadata" actions
      print_entry_json "${name}" '"list" "exec" "metadata"'
  }

  to_json_array "`print_vm_json ${SSHFS_VM_ONE}` `print_vm_json ${SSHFS_VM_TWO}`"
  exit 0
fi

# path is of the form /<vm>/... so get the VM's name
vm=`get_root ${path}`

path=`strip_root ${path}`
if [[ "${path}" == "" ]]; then
  # Our action's being invoked on a VM. Since a VM only supports the
  # "list", "exec" and "metadata" actions, we case our code on those
  # actions
  case "${action}" in
  "list")
    # "list"'ing a VM is equivalent to listing its root
    print_children ${vm} "/"
    exit 0
  ;;
  "exec")
    cmd="$3"

    shift
    shift
    shift
    args="$@"

    # exec'ing <cmd> <args> on a VM is equivalent to invoking them
    # on the VM via. ssh (vm_exec)
    #
    # TODO: Handle stdin
    vm_exec "${vm}" "${cmd} ${args}"
    exit "$?"
  ;;
  "metadata")
    # We could provide more metadata here, such as the VM's platform,
    # architecture, and processor information. However, the example
    # below is good enough for Wash.
    echo "{\
\"provider\":\"vmware\"\
}"
    exit 0
  ;;
  *)
    # Notice how we print errors to stderr then exit with a non-zero
    # exit code. This tells Wash that our invocation failed.
    echo "missing a case statement for the ${action} action" >2
    exit 1
  ;;
  esac
fi

# Our path is an absolute path in the VM's filesystem.
# Thus, we can just case on all the possible actions that can
# be passed-in.
case "${action}" in
"list")
  print_children ${vm} "${path}/"
  exit 0
;;
"read")
  vm_exec "${vm}" "cat ${path}"
  exit 0
;;
"stream")
  # Notice how we print the header first before anything else.
  # This way, Wash knows that we're about to stream some data.
  echo "200"

  # We could also `cat` here, which is useful for large files.
  # Instead, we choose `tail -f` to show that external plugins
  # can implement their own `tail -f` like behavior. Don't worry,
  # Wash will send the SIGTERM signal to our process when it no
  # longer needs our streamed data.
  vm_exec "${vm}" "tail -f ${path}"
  exit 0
;;
*)
  echo "missing a case statement for the ${action} action" >2
  exit 1
;;
esac