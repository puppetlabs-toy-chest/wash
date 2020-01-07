#!/usr/bin/env bash

script="$1"
if [[ -z "${script}" ]]; then
  echo "USAGE: ${0} <script>"
  exit 1
fi
width="60"
height="15"

# Strip the extension
script_name=`basename "${script%%.*}"`

# Record the screencast
cast_path="`mktemp -d`/${script_name}.cast"
echo "Recording the screencast... Start typing like mad as soon as you see the wash prompt."
sleep 2
asciinema rec -c "doitlive play ${script} -q" "${cast_path}"
echo "In a separate terminal, double-check the recorded screencast by invoking"
echo "  asciinema play ${cast_path}"
echo ""
echo -n "Does the recorded screencast look OK (y/n)? "
read confirmation
if [[ ! ${confirmation} =~ ^[yY] ]]; then
  echo "Screencast not OK. Edit your script (${script}), then re-run"
  echo "  ${0} ${script}"
  exit 1
fi

# Edit the dimensions
sed -i '' -E "s/(.*)\"width\": [0-9]+(.*)/\1\"width\": ${width}\2/g" ${cast_path}
sed -i '' -E "s/(.*)\"height\": [0-9]+(.*)/\1\"height\": ${height}\2/g" ${cast_path}

# Move the recorded screencast over to its actual location. Note that
# the script's path <docs_dir>/_screencasts/<category>/<script_name>.sh
docs_dir=$(dirname $(dirname $(dirname ${script})))
category=$(basename $(dirname ${script}))
cast_dir="${docs_dir}/assets/screencasts/${category}"
mkdir -p "${cast_dir}"
mv ${cast_path} "${cast_dir}"
echo "The cast has been saved to ${cast_dir}/`basename ${cast_path}`"
echo "Don't forget to set/reset the poster"
