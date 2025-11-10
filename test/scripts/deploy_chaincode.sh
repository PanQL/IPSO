#!/bin/bash

CC_PACKAGE_ID=''

#TODO extremely slow
package_chaincode() {
  infoln "Packaging chaincode"
  set -x
  peer lifecycle chaincode package ${CC_DST_PATH} --path ${CC_SRC_PATH} --lang ${CC_LANGUAGE} --label ${CC_NAME}_${CC_VERSION}
  res=$?
  { set +x; } 2>/dev/null

  verify_result $res "Chaincode is packaged" "Failed to package chaincode"
}

# install_chaincode ORG PEER
install_chaincode() {
  local ORG=$1
  local PEER=$2
  local peer_fullname="org${ORG}peer${PEER}"

  use_peer ${ORG} ${PEER}

  infoln "Installing chaincode on peer '${peer_fullname}'"
  set -x
  peer lifecycle chaincode install ${CC_DST_PATH}
  res=$?
  { set +x; } 2>/dev/null

  verify_result $res "Chaincode is installed on peer '${peer_fullname}'" "Failed to install chaincode on peer '${peer_fullname}'"
}

# query_installed ORG PEER
query_installed() {
  local ORG=$1
  local PEER=$2
  local peer_fullname="org${ORG}peer${PEER}"

  use_peer ${ORG} ${PEER}

  infoln "Querying chaincode installed on ${peer_fullname}"
  set -x
  peer lifecycle chaincode queryinstalled >log.txt
  res=$?
  { set +x; } 2>/dev/null

  CC_PACKAGE_ID=$(sed -n "/${CC_NAME}_${CC_VERSION}/{s/^Package ID: //; s/, Label:.*$//; p;}" log.txt)

  # cat log.txt
  rm log.txt

  verify_result $res "Query chaincode installed on peer '${peer_fullname}'. Package ID: ${CC_PACKAGE_ID}" "Failed to query chaincode installed on peer ${peer_fullname}"
}

# approve_for_my_org ORG PEER
approve_for_my_org() {
  local ORG=$1
  local PEER=$2
  local peer_fullname="org${ORG}peer${PEER}"

  use_peer ${ORG} ${PEER}

  infoln "Approving chaincode definition by peer '${peer_fullname}' on channel '${APP_CHANNEL_NAME}'"
  set -x
  peer lifecycle chaincode approveformyorg -o ${ORDERER_ENDPOINT} --ordererTLSHostnameOverride ${ORDERER_HOSTNAME} --tls --cafile ${ORG0_CA} --channelID ${APP_CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --package-id ${CC_PACKAGE_ID} --sequence 1
  res=$?
  { set +x; } 2>/dev/null

  verify_result $res "Chaincode definition is approved by peer '${peer_fullname}' on channel '${APP_CHANNEL_NAME}'" "Failed to approve chaincode definition by peer '${peer_fullname}' on channel '${APP_CHANNEL_NAME}'"
}

# check_commit_readiness ORG PEER ORG1_READY ORD2_READY
check_commit_readiness() {
  local ORG=$1
  local PEER=$2
  local ORG1_READY=$3
  local ORG2_READY=$4
  local peer_fullname="org${ORG}peer${PEER}"

  use_peer ${ORG} ${PEER}

  infoln "Checking commit readiness of the chaincode definition on '${peer_fullname}' on channel '${APP_CHANNEL_NAME}'..."

  local rc=1
  local COUNTER=0
  while ((rc != 0 && COUNTER < MAX_RETRY)); do
    sleep ${DELAY}

    infoln "Attempting to check chaincode commit readiness on peer '${peer_fullname}', retry after ${DELAY} seconds."

    set -x
    peer lifecycle chaincode checkcommitreadiness --channelID ${APP_CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --sequence 1 --tls --cafile ${ORG0_CA} --output json >log.txt
    res=$?
    { set +x; } 2>/dev/null

    if ((res == 0)); then
      rc=0
      if [[ ${DEPLOY_SCHEME} == 1p1o ]]; then
        grep "\"Org1MSP\": ${ORG1_READY}" log.txt &>/dev/null || rc=1
      elif [[ ${DEPLOY_SCHEME} == 4p1o ]]; then
        grep "\"Org1MSP\": ${ORG1_READY}" log.txt &>/dev/null || rc=1
        grep "\"Org2MSP\": ${ORG2_READY}" log.txt &>/dev/null || rc=1
      fi
    fi

    rm log.txt

    ((COUNTER = COUNTER + 1))
  done

  verify_result $rc "Check commit readiness of the chaincode definition on peer '${peer_fullname}' on channel '${APP_CHANNEL_NAME}'" "Failed to check commit readiness of the chaincode definition on '${peer_fullname}' after ${MAX_RETRY} attempts."
}

# commit_chaincode_definition ORG PEER
commit_chaincode_definition() {
  local ORG=$1
  local PEER=$2
  local peer_fullname="org${ORG}peer${PEER}"

  use_peer ${ORG} ${PEER}

  local PEER_CONN_PARMS=""
  if [[ ${DEPLOY_SCHEME} == 1p1o ]]; then
    PEER_CONN_PARMS="--peerAddresses org1peer0:7051 --tlsRootCertFiles ${ORG1_CA}"
  elif [[ ${DEPLOY_SCHEME} == 4p1o ]]; then
    PEER_CONN_PARMS="--peerAddresses org1peer0:7051 --tlsRootCertFiles ${ORG1_CA}"
    PEER_CONN_PARMS="${PEER_CONN_PARMS} --peerAddresses org1peer1:7051 --tlsRootCertFiles ${ORG1_CA}"
    PEER_CONN_PARMS="${PEER_CONN_PARMS} --peerAddresses org2peer0:7051 --tlsRootCertFiles ${ORG2_CA}"
    PEER_CONN_PARMS="${PEER_CONN_PARMS} --peerAddresses org2peer1:7051 --tlsRootCertFiles ${ORG2_CA}"
  fi

  infoln "Committing chaincode definition by peer '${peer_fullname}' on channel '${APP_CHANNEL_NAME}'"
  set -x
  peer lifecycle chaincode commit -o ${ORDERER_ENDPOINT} --ordererTLSHostnameOverride ${ORDERER_HOSTNAME} --channelID ${APP_CHANNEL_NAME} --name ${CC_NAME} --version ${CC_VERSION} --sequence 1 --tls --cafile ${ORG0_CA} ${PEER_CONN_PARMS}
  res=$?
  { set +x; } 2>/dev/null

  verify_result $res "Chaincode definition is committed by peer '${peer_fullname}' on channel '${APP_CHANNEL_NAME}'" "Failed to commit chaincode definition by peer '${peer_fullname}' on channel '${APP_CHANNEL_NAME}'"
}

# query_committed ORG PEER
query_committed() {
  local ORG=$1
  local PEER=$2
  local peer_fullname="org${ORG}peer${PEER}"
  local expected_result="Version: ${CC_VERSION}, Sequence: 1, Endorsement Plugin: escc, Validation Plugin: vscc"

  use_peer ${ORG} ${PEER}

  infoln "Querying chaincode definition on peer ${peer_fullname} on channel '${APP_CHANNEL_NAME}'..."

  local rc=1
  local COUNTER=0
  while ((rc != 0 && COUNTER < MAX_RETRY)); do
    sleep ${DELAY}

    infoln "Attempting to query committed status on peer '${peer_fullname}', retry after ${DELAY} seconds."

    set -x
    peer lifecycle chaincode querycommitted --channelID ${APP_CHANNEL_NAME} --name ${CC_NAME} >log.txt
    res=$?
    { set +x; } 2>/dev/null

    if ((res == 0)); then
      local VALUE=$(cat log.txt | grep -o '^Version: '$CC_VERSION', Sequence: [0-9]*, Endorsement Plugin: escc, Validation Plugin: vscc')
    fi

    if [[ ${VALUE} == ${expected_result} ]]; then
      ((rc = 0))
    fi

    ((COUNTER = COUNTER + 1))
  done

  rm log.txt

  verify_result $rc "Query chaincode definition by peer '${peer_fullname}' on channel '${APP_CHANNEL_NAME}'" "Failed to query chaincode definition by '${peer_fullname}' after ${MAX_RETRY} attempts."
}

# package_chaincode

if [[ ${DEPLOY_SCHEME} == 1p1o ]]; then
  infoln "Installing chaincode on org1"
  install_chaincode 1 0

  infoln "Querying whether the chaincode is installed"
  query_installed 1 0

  approve_for_my_org 1 0
  check_commit_readiness 1 0 true false

  infoln "Commit the definition"
  commit_chaincode_definition 1 0

  infoln "Query on all peers to see that the definition committed successfully"
  query_committed 1 0

elif [[ ${DEPLOY_SCHEME} == 4p1o ]]; then
  infoln "Installing chaincode on org1"
  install_chaincode 1 0
  install_chaincode 1 1
  infoln "Installing chaincode on org2"
  install_chaincode 2 0
  install_chaincode 2 1

  infoln "Querying whether the chaincode is installed"
  query_installed 1 0
  query_installed 1 1
  query_installed 2 0
  query_installed 2 1

  approve_for_my_org 1 0
  check_commit_readiness 1 0 true false
  check_commit_readiness 1 1 true false

  approve_for_my_org 2 0
  check_commit_readiness 2 0 true true
  check_commit_readiness 2 1 true true

  infoln "Commit the definition"
  commit_chaincode_definition 1 0

  infoln "Query on all peers to see that the definition committed successfully"
  query_committed 1 0
  query_committed 1 1
  query_committed 2 0
  query_committed 2 1
fi
