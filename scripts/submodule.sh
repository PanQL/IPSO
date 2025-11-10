#!/bin/bash

# Get the folder path of this file
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

# Import functions
source ${SCRIPT_DIR}/print.sh

function pull_submodule() {
  infoln "Pulling submodules"
  git submodule init && git submodule update
  if [[ $? -ne 0 ]]; then
    fatalln "Fail to pull submodules."
  fi
}
