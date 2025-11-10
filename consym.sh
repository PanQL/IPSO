#!/bin/bash

####################
# Import variables
####################
source ./env.sh

####################
# Import functions
####################
source scripts/image.sh
source scripts/print.sh
source scripts/help.sh
source scripts/submodule.sh

####################
# Functions
####################
function build_images() {
  if [[ ${BUILD_HLF_PEER_IMAGE} == 'true' ]]; then
    infoln "Building HLF peer image"
    build_hlf_peer_image
    save_hlf_peer_image
  fi

  if [[ ${BUILD_HLF_ORDERER_IMAGE} == 'true' ]]; then
    infoln "Building HLF orderer image"
    build_hlf_orderer_image
    save_hlf_orderer_image
  fi

  if [[ ${BUILD_HLF_TOOLS_IMAGE} == 'true' ]]; then
    infoln "Building HLF tools image"
    build_hlf_tools_image
    save_hlf_tools_image
  fi

  if [[ ${BUILD_TAPE_IMAGE} == 'true' ]]; then
    infoln "Building tape image"
    build_tape
    save_tape_image
  fi
}

function distribute_images() {
  #TODO
  distribute_hlf_images
}

function run() {
  cd ${TEST_PATH}
  ./test.sh run
}

function analyze() {
  cd ${TEST_PATH}
  ./test.sh analyze
}

function clean() {
  if [[ ${CLEAN_HLF_PEER_IMAGE} == 'true' ]]; then
    infoln "Cleaning HLF peer image"
    clean_hlf_peer
  fi

  if [[ ${CLEAN_HLF_ORDERER_IMAGE} == 'true' ]]; then
    infoln "Cleaning HLF orderer image"
    clean_hlf_orderer
  fi

  if [[ ${CLEAN_HLF_TOOLS_IMAGE} == 'true' ]]; then
    infoln "Cleaning HLF tools image"
    clean_hlf_tools
  fi

  if [[ ${CLEAN_TAPE_IMAGE} == 'true' ]]; then
    infoln "Cleaning tape image"
    clean_tape
  fi

  if [[ ${CLEAN_DANGLING_IMAGE} == 'true' ]]; then
    docker image prune --force
  fi

  if [[ ${CLEAN_RESULT} == 'true' ]]; then
    cd ${TEST_PATH} && ./clean.sh
  fi
}

####################
# CLI options
####################
CMD=$1
shift
while [[ $# -ge 1 ]]; do
  key="$1"
  case ${key} in
  '-h' | '--help')
    print_help ${CMD}
    shift
    ;;
  *)
    #TODO
    print_help
    ;;
  esac
done

####################
# Main
####################
case ${CMD} in
'build')
  build_images
  ;;
'distribute')
  distribute_images
  ;;
'run')
  run
  ;;
'analyze')
  analyze
  ;;
'clean')
  clean
  ;;
esac
