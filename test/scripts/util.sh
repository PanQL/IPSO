#!/bin/bash

export C_RESET='\033[0m'
export C_RED='\033[0;31m'
export C_GREEN='\033[0;32m'
export C_BLUE='\033[0;34m'
export C_YELLOW='\033[1;33m'

# println echos string
function println() {
  echo -e "$1"
}

# errorln echos i red color
function errorln() {
  println "${C_RED}${1}${C_RESET}"
}

# successln echos in green color
function successln() {
  println "${C_GREEN}${1}${C_RESET}"
}

# infoln echos in blue color
function infoln() {
  println "${C_BLUE}${1}${C_RESET}"
}

# warnln echos in yellow color
function warnln() {
  println "${C_YELLOW}${1}${C_RESET}"
}

# fatalln echos in red color and exits with fail status
function fatalln() {
  errorln "$1"
  exit 1
}

# verify_result <exit code> <success message> <failure message>
function verify_result() {
  if (($# != 3)); then
    fatalln "verify_result does not get 3 arguments"
  fi

  if (($1 == 0)); then
    successln "$2"
  else
    fatalln "$3"
  fi
}

# use_peer org peer
function use_peer() {
  local ORG=$1
  local PEER=$2
  infoln "Using org${ORG}peer${PEER}"

  export CORE_PEER_LOCALMSPID="Org${ORG}MSP"
  export CORE_PEER_MSPCONFIGPATH=${PWD}/organization/org${ORG}/user/org${ORG}admin@org${ORG}/msp
  export CORE_PEER_ADDRESS=org${ORG}peer${PEER}:7051
  local ca_name="ORG${ORG}_CA"
  export CORE_PEER_TLS_ROOTCERT_FILE=${!ca_name}
}

export -f println
export -f errorln
export -f successln
export -f infoln
export -f warnln
export -f fatalln
export -f verify_result
export -f use_peer
