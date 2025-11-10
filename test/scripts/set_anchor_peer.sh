#!/bin/bash

source ${SCRIPT_PATH}/util.sh
source ${SCRIPT_PATH}/config_update.sh

ORG=$1
PEER=$2
CHANNEL_NAME=$3
PEER_FULLNAME="org${ORG}peer${PEER}"

create_anchor_peer_update() {
  local config_update=$1

  local original_config_json=${CORE_PEER_LOCALMSPID}config.json
  local modified_config_json=${CORE_PEER_LOCALMSPID}modified_config.json

  infoln "Fetching channel config for channel ${CHANNEL_NAME}"
  fetch_channel_config ${ORG} ${PEER} ${CHANNEL_NAME} ${original_config_json}

  infoln "Generating anchor peer update transaction for Org${ORG} on channel '${CHANNEL_NAME}'"
  HOST="org${ORG}peer${PEER}"
  PORT=7051
  set -x
  # Modify the configuration to append the anchor peer
  jq '.channel_group.groups.Application.groups.'${CORE_PEER_LOCALMSPID}'.values += {"AnchorPeers":{"mod_policy": "Admins","value":{"anchor_peers": [{"host": "'${HOST}'","port": '${PORT}'}]},"version": "0"}}' ${original_config_json} >${modified_config_json}
  { set +x; } 2>/dev/null

  create_config_update ${CHANNEL_NAME} ${original_config_json} ${modified_config_json} ${config_update}

  rm ${original_config_json} ${modified_config_json}
}

update_anchor_peer() {
  local config_update=${CORE_PEER_LOCALMSPID}anchors.tx

  infoln "Updating ${PEER_FULLNAME} as an anchor peer"

  create_anchor_peer_update ${config_update}

  set -x
  peer channel update -o ${ORDERER_ENDPOINT} --ordererTLSHostnameOverride ${ORDERER_HOSTNAME} -c ${CHANNEL_NAME} -f ${config_update} --tls --cafile ${ORG0_CA}
  res=$?
  { set +x; } 2>/dev/null

  rm ${config_update}

  verify_result $res "Update ${PEER_FULLNAME} as an anchor peer" "Failed to update ${PEER_FULLNAME} as an anchor peer"
}

use_peer ${ORG} ${PEER}

update_anchor_peer
