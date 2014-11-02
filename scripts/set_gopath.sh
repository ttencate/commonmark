#!/bin/bash

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  echo 'This script must be sourced, not executed. Run:'
  echo
  echo '    . scripts/set_gopath.sh'
  echo
  exit 1
fi

cd ${BASH_SOURCE[0]}/..
export GOPATH="$GOPATH:$(pwd)"

echo 'Updated GOPATH in the current shell:'
echo
echo "    GOPATH=$GOPATH"
echo
