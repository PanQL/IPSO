#!/bin/bash

source ./scripts/util.sh

# fetch_channel_config <org> <peer> <channel_id> <output_json>
# Writes the current channel config for a given channel to a JSON file
# NOTE: this must be run in a CLI container since it requires configtxlator
fetch_channel_config() {
  local org=$1
  local peer=$2
  local channel=$3
  local output_file=$4
  local config_block_pb=config_block.pb

  use_peer ${org} ${peer}

  infoln "Fetching the most recent configuration block for the channel"
  set -x
  peer channel fetch config ${config_block_pb} -o ${ORDERER_ENDPOINT} --ordererTLSHostnameOverride ${ORDERER_HOSTNAME} -c ${channel} --tls --cafile ${ORG0_CA}
  { set +x; } 2>/dev/null

  infoln "Decoding config block to JSON and isolating config to ${output_file}"
  set -x
  configtxlator proto_decode --input ${config_block_pb} --type common.Block | jq .data.data[0].payload.data.config >${output_file}
  { set +x; } 2>/dev/null

  rm ${config_block_pb}
}

# create_config_update <channel_id> <original_config.json> <modified_config.json> <output.pb>
# Takes an original and modified config, and produces the config update tx
# which transitions between the two
# NOTE: this must be run in a CLI container since it requires configtxlator
create_config_update() {
  local channel=$1
  local original_config_json=$2
  local modified_config_json=$3
  local output_file=$4

  local original_config_pb='original_config.pb'
  local modified_config_pb='modified_config.pb'
  local config_update_pb='config_update.pb'
  local config_update_json='config_update.json'
  local config_update_envelope='config_update_in_envelope.json'

  set -x
  configtxlator proto_encode --input ${original_config_json} --type common.Config >${original_config_pb}
  configtxlator proto_encode --input ${modified_config_json} --type common.Config >${modified_config_pb}
  configtxlator compute_update --channel_id "${channel}" --original ${original_config_pb} --updated ${modified_config_pb} >${config_update_pb}
  configtxlator proto_decode --input ${config_update_pb} --type common.ConfigUpdate >${config_update_json}
  echo '{"payload":{"header":{"channel_header":{"channel_id":"'${channel}'", "type":2}},"data":{"config_update":'$(cat ${config_update_json})'}}}' | jq . >${config_update_envelope}
  configtxlator proto_encode --input ${config_update_envelope} --type common.Envelope >${output_file}
  { set +x; } 2>/dev/null

  rm ${original_config_pb} ${modified_config_pb} ${config_update_pb} ${config_update_json} ${config_update_envelope}
}
