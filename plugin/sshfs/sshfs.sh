#!/usr/bin/env bash

## HELPERS

function get_root {
  local path="$1"

  echo "${path}" | sed -E 's:.*^/([^/]+).*:\1:'
}

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

function print_entry_json() {
  local name="$1"
  local supported_actions="$2"

  local attributes_json="$3"
  if [[ -z "${attributes_json}" ]]; then
    attributes_json="{}"
  fi

  # TODO: Include attributes for files
  local supported_actions_json=`to_json_array "${supported_actions}"`

  echo "{\
\"name\":\"${name}\",\
\"supported_actions\":${supported_actions_json},\
\"attributes\":${attributes_json}\
}"
}

function print_file_json {
  local vm="$1"
  local isDir="$2"
  local size="$3"
  local atime="$4"
  local mtime="$5"
  local ctime="$6"
  local mode="$7"
  local path="$8"

  name=`basename "${path}"`

  if [[ ${isDir} -eq 0 ]]; then
    supported_actions='"list"'
  else
    supported_actions='"read" "stream"'
  fi
  
  attributes_json=$(echo "{\
\"Atime\":${atime},\
\"Mtime\":${mtime},\
\"Ctime\":${ctime},\
\"Mode\":$(echo $((16#${mode}))),\
\"Size\":${size}\
}")

  print_entry_json "${name}" "${supported_actions}" "${attributes_json}"
}

function list_children {
  local vm="$1"
  local dir="$2"

  # Each line of stat_output is a child of ${dir}, and has the following format:
  #   <is_dir> <sizeAttr> <atime> <mtime> <ctime> <mode> <path>
  stat_output=`vm_exec ${vm} "find ${dir} -mindepth 1 -maxdepth 1 | xargs -n1 -r -I {} bash -c '(test -d \\$@; echo -n \"\\$? \") && stat -c \"%s %X %Y %Z %f %n\" \\$@' _ {}"`
  if [[ -z "${stat_output}" ]]; then
    echo "[]"
    return 0
  fi

  # Now we parse each line of stat_output into its corresponding
  # entry JSON object, and stream the results to stdout. After this
  # code runs, stdout should be something like:
  #
  # [
  # <child_json>,
  # <child_json>,
  # ...
  # <child_json> 
  # ]

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
       | sed "s/^/${vm} /"\
       | xargs -n8 -P 10 -I {} bash -c 'print_file_json $@' _ {}\
       | sed s/$/,/
  fi

  # Now print the last child
  echo "${stat_output}"\
     | tail -n1\
     | print_file_json "${vm}" ${stat_output}

  echo "]"
}

##

# FS is modeled as: /sshfs/<VM>/<VM_FS...>

action="$1"
if [[ "${action}" == "init" ]]; then
  print_entry_json "sshfs" '"list"'
  exit 0
fi

path="$2"

path=`strip_root ${path}`
if [[ "${path}" == "" ]]; then
  # Our action's being invoked on the root. Since Wash only passes
  # in supported actions, and since our root only supports the
  # "list" action, we can assume that action == "list" here.
  function print_vm_json() {
      local name="$1"

      print_entry_json "${name}" '"list" "exec" "metadata"'
  }

  to_json_array "`print_vm_json ${SSHFS_RHEL_HOST}` `print_vm_json ${SSHFS_DEBIAN_HOST}`"
  exit 0
fi

# path is of the form /<vm>/...
vm=`get_root ${path}`

path=`strip_root ${path}`
if [[ "${path}" == "" ]]; then
  # Our action's being invoked on a VM.
  case "${action}" in
  "list")
    list_children ${vm} "/"
    exit 0
  ;;
  "exec")
    cmd="$3"

    shift
    shift
    shift
    args="$@"

    vm_exec "${vm}" "${cmd} ${args}"
    exit "$?"
  ;;
  "metadata")
    echo "{\
\"provider\":\"vmware\"\
}"
    exit 0
  ;;
  *)
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
  list_children ${vm} "${path}/"
  exit 0
;;
"read")
  vm_exec "${vm}" "cat ${path}"
  exit 0
;;
"stream")
  echo "200"

  vm_exec "${vm}" "cat ${path}"
  exit 0
;;
*)
  echo "missing a case statement for the ${action} action" >2
  exit 1
;;
esac