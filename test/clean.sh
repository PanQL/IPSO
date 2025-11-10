#!/bin/bash

source ./env.sh
source ${SCRIPT_PATH}/util.sh

infoln "Removing artifacts"
sudo -E docker run --rm -v ${PWD}:/data busybox sh -c 'cd /data && rm -rf ca organization channel result hlf_org0orderer.log hlf_org1peer0.log'