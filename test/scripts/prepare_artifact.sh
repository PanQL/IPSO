#!/bin/bash

source ./env.sh
source ${SCRIPT_PATH}/util.sh

function create_organizations() {
  rm -rf organization ca channel &>/dev/null
  mkdir organization ca channel

  infoln "Generating certificates using Fabric CA"

  sudo -E docker compose -f ${CA_COMPOSE_FILE} up -d 2>&1

  while :; do
    if [[ ${DEPLOY_SCHEME} == 1p1o ]]; then
      if [[ ! -f "./ca/org1/tls-cert.pem" || ! -f "./ca/org0/tls-cert.pem" ]]; then
        sleep 1
      else
        break
      fi

    elif [[ ${DEPLOY_SCHEME} == 4p1o ]]; then
      if [[ ! -f "./ca/org1/tls-cert.pem" || ! -f "./ca/org2/tls-cert.pem" || ! -f "./ca/org0/tls-cert.pem" ]]; then
        sleep 1
      else
        break
      fi
    fi

  done

  ${SCRIPT_PATH}/register_enroll.sh

  # Bring down the CA once all certificates are issued
  sudo -E docker compose -f ${CA_COMPOSE_FILE} down --timeout 3
}

function create_consortium() {
  which configtxgen &>/dev/null || fatalln "configtxgen tool not found."

  infoln "Generating orderer genesis block"
  mkdir -p ./channel/${SYSTEM_CHANNEL_NAME}
  set -x
  configtxgen -profile TwoOrgsOrdererGenesis -channelID ${SYSTEM_CHANNEL_NAME} -outputBlock ./channel/${SYSTEM_CHANNEL_NAME}/genesis.block
  res=$?
  { set +x; } 2>/dev/null

  verify_result $res "Generate orderer genesis block" "Failed to generate orderer genesis block"
}

export FABRIC_CFG_PATH=${CONFIG_PATH}

create_organizations

create_consortium
