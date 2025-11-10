#!/bin/bash

####################
# Configuration
####################
export DEPLOY_MODE='swarm'  # 'compose' 'swarm'
export DEPLOY_SCHEME='4p1o' # '4p1o' = 4 peers + 1 orderer; '1p1o' = 1 peer + 1 orderer;
export SCRIPT_PATH="${PWD}/scripts"
export CONFIG_PATH="${PWD}/config/${DEPLOY_SCHEME}"

export HLF_COMPOSE_FILE="${CONFIG_PATH}/hlf-compose.yaml"
export HLF_SWARM_FILE="${CONFIG_PATH}/hlf-swarm.yaml"
export CA_COMPOSE_FILE="${CONFIG_PATH}/ca-compose.yaml"

export PROJECT_NAME='hlf'
export NETWORK_NAME='hlf'

export SYSTEM_CHANNEL_NAME='system-channel'
export APP_CHANNEL_NAME='mychannel'
export APP_CHANNEL_PATH="${PWD}/channel/${APP_CHANNEL_NAME}"
export TX_FILE="${APP_CHANNEL_PATH}/${APP_CHANNEL_NAME}.tx"
export BLOCK_FILE="${APP_CHANNEL_PATH}/${APP_CHANNEL_NAME}.block"

export CC_NAME='smallbank'
export CC_LANGUAGE='golang'
export CC_VERSION='1.0'
export CC_SRC_PATH="${PWD}/chaincode/src/smallbank"
# the label must be 'smallbank_1.0'
export CC_DST_PATH="${PWD}/chaincode/build/${HLF_TYPE}_${CC_NAME}.tar.gz"
# export CC_DST_PATH="${PWD}/chaincode/build/target_chaincode.tar.gz"

export MAX_RETRY=5
export DELAY=3

export TAPE_CONFIG_FILE_PUT="./config/${DEPLOY_SCHEME}/tape-put.yaml"
export TAPE_CONFIG_FILE_CONFLICT="./config/${DEPLOY_SCHEME}/tape-conflict.yaml"

export RESULT_PATH='result'
export RESULT_LOG="${RESULT_PATH}/tx.log"
export LATENCY_REPORT="${RESULT_PATH}/latency.csv"
export LATENCY_FIG="${RESULT_PATH}/latency.pdf"
export CONFLICT_REPORT="${RESULT_PATH}/conflict.txt"
export TAPE_STDOUT_FILE_PUT="${RESULT_PATH}/put_stdout.txt"
export TAPE_STDERR_FILE_PUT="${RESULT_PATH}/put_stderr.txt"
export TAPE_STDOUT_FILE_CONFLICT="${RESULT_PATH}/conflict_stdout.txt"
export TAPE_STDERR_FILE_CONFLICT="${RESULT_PATH}/conflict_stderr.txt"

export CORE_PEER_TLS_ENABLED='true'
export ORG0_CA="${PWD}/organization/org0/tlsca/tlsca.org0-cert.pem"
export ORG1_CA="${PWD}/organization/org1/tlsca/tlsca.org1-cert.pem"
export ORG2_CA="${PWD}/organization/org2/tlsca/tlsca.org2-cert.pem"
export ORDERER_HOSTNAME='org0orderer'
export ORDERER_ENDPOINT="${ORDERER_HOSTNAME}:7050"
