#!/bin/bash

# PEER_OLD_NAME="hyperledger/${HLF_IMAGE_PREFIX}-peer:${HLF_OLD_TAG}"
PEER_NAME="hyperledger/${HLF_IMAGE_PREFIX}-peer:latest"
PEER_IMAGE_FILE_NAME="${HLF_IMAGE_PREFIX}-peer:latest.tar"
PEER_IMAGE_FILE="${IMAGE_PATH}/${PEER_IMAGE_FILE_NAME}"

# ORDERER_OLD_NAME="hyperledger/${HLF_IMAGE_PREFIX}-orderer:${HLF_OLD_TAG}"
ORDERER_NAME="hyperledger/${HLF_IMAGE_PREFIX}-orderer:latest"
ORDERER_IMAGE_FILE_NAME="${HLF_IMAGE_PREFIX}-orderer:latest.tar"
ORDERER_IMAGE_FILE="${IMAGE_PATH}/${ORDERER_IMAGE_FILE_NAME}"

# TOOLS_OLD_NAME="hyperledger/${HLF_IMAGE_PREFIX}-tools:${HLF_OLD_TAG}"
TOOLS_NAME="hyperledger/${HLF_IMAGE_PREFIX}-tools:latest"
TOOLS_IMAGE_FILE_NAME="${HLF_IMAGE_PREFIX}-tools:latest.tar"
TOOLS_IMAGE_FILE="${IMAGE_PATH}/${TOOLS_IMAGE_FILE_NAME}"

TAPE_OLD_NAME="tape:latest"
TAPE_NEW_NAME="tape:${HLF_TYPE}"
TAPE_IMAGE_FILE_NAME="${TAPE_NEW_NAME}.tar"
TAPE_IMAGE_FILE="${IMAGE_PATH}/${TAPE_IMAGE_FILE_NAME}"

function get_go_tags() {
  case "${HLF_TYPE}" in
  vanilla | fast | sharp | consym)
    GO_TAGS="${HLF_TYPE}"
    ;;
  strawman)
    GO_TAGS="redis"
    ;;
  *)
    echo "Invalid HLF build option"
    exit 1
    ;;
  esac
}

function build_hlf_peer_image() {
  cd ${HLF_PATH}
  get_go_tags || exit 1
  sudo make HLF_VARAINT="${HLF_TYPE}" GO_TAGS="${GO_TAGS}" peer-docker
}

function build_hlf_orderer_image() {
  cd ${HLF_PATH}
  get_go_tags || exit 1
  sudo make HLF_VARAINT="${HLF_TYPE}" GO_TAGS="${GO_TAGS}" orderer-docker
}

function build_hlf_tools_image() {
  cd ${HLF_PATH}
  get_go_tags || exit 1
  sudo make HLF_VARAINT="${HLF_TYPE}"  GO_TAGS="${GO_TAGS}" tools-docker
}

function build_tape() {
  cd ${TAPE_PATH}

  # To fetch from private Github repo, a user name and a PAT must be supplied
  GITHUB_USER=
  GITHUB_PAT=
  source ./github_credential

  if [[ ! -f ./Dockerfile ]]; then
    echo "No Dockerfile in current directory. Exit."
    exit 1
  fi

  docker build --no-cache -f Dockerfile -t tape --build-arg GITHUB_USER=${GITHUB_USER} --build-arg GITHUB_PAT=${GITHUB_PAT} .
  docker tag ${TAPE_OLD_NAME} ${TAPE_NEW_NAME}
  if [[ $? -ne 0 ]]; then
    echo "Fail to build tape docker image."
    exit 1
  fi
}

function clean_hlf_peer() {
  cd ${HLF_PATH}
  sudo make HLF_VARAINT="${HLF_TYPE}" peer-docker-clean
}

function clean_hlf_orderer() {
  cd ${HLF_PATH}
  sudo make HLF_VARAINT="${HLF_TYPE}" orderer-docker-clean
}

function clean_hlf_tools() {
  cd ${HLF_PATH}
  sudo make HLF_VARAINT="${HLF_TYPE}" tools-docker-clean
}

function clean_tape() {
  cd ${TAPE_PATH}
  docker image rm --force $(sudo docker image ls "tape" -q | sort | uniq) 2>/dev/null # Remove other tape images
}

function save_hlf_peer_image() {
  mkdir -p ${IMAGE_PATH} 2>/dev/null
  docker save -o ${PEER_IMAGE_FILE} ${PEER_NAME}
}

function save_hlf_orderer_image() {
  mkdir -p ${IMAGE_PATH} 2>/dev/null
  docker save -o ${ORDERER_IMAGE_FILE} ${ORDERER_NAME}
}

function save_hlf_tools_image() {
  mkdir -p ${IMAGE_PATH} 2>/dev/null
  docker save -o ${TOOLS_IMAGE_FILE} ${TOOLS_NAME}
}

function save_tape_image() {
  mkdir -p ${IMAGE_PATH} 2>/dev/null
  docker save -o ${TAPE_IMAGE_FILE} ${TAPE_NEW_NAME}
}

function remove_image_files() {
  rm ${PEER_IMAGE_FILE} ${ORDERER_IMAGE_FILE} ${TOOLS_IMAGE_FILE} 2>/dev/null
}

function distribute_hlf_images() {
  for host in ${HOSTS[@]}; do
    # Skip the current host
    ip a | grep "$host" >/dev/null
    if [[ $? == 0 ]]; then
      continue
    fi

    target="$USER@$host"
    target_path="~/consym"

    # Copy the image tar files to the target host's path
    infoln "Transfering images to '$host'"
    scp ${PEER_IMAGE_FILE} ${ORDERER_IMAGE_FILE} ${TOOLS_IMAGE_FILE} ${TAPE_IMAGE_FILE} ${target}:${target_path}

    infoln "Loading images to '$host'"
    set -x
    ssh ${target} "docker rmi --force ${PEER_NAME} ${ORDERER_NAME} ${TOOLS_NAME} ${TAPE_NEW_NAME} 2>/dev/null" || true
    ssh ${target} "cd ${target_path} && docker load --input ${PEER_IMAGE_FILE_NAME} && docker load --input ${ORDERER_IMAGE_FILE_NAME} && docker load --input ${TOOLS_IMAGE_FILE_NAME} && docker load --input ${TAPE_IMAGE_FILE_NAME}"
    # ssh ${target} "cd ${target_path} && sudo docker load --input ${PEER_IMAGE_FILE_NAME} && sudo docker load --input ${ORDERER_IMAGE_FILE_NAME} && sudo docker load --input ${TOOLS_IMAGE_FILE_NAME} && sudo docker load --input ${TAPE_IMAGE_FILE_NAME}"
    set +x
  done
}
