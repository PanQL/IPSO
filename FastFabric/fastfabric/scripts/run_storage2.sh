#!/bin/bash
source base_parameters.sh

# 使用 FAST_PEER_ADDRESS 而不是 hostname
p_addr=$(get_correct_peer_address ${FAST_PEER_ADDRESS})

export FABRIC_LOGGING_SPEC=INFO
# export CORE_PEER_MSPCONFIGPATH=${FABRIC_CFG_PATH}/crypto-config/peerOrganizations/${PEER_DOMAIN}/peers/${p_addr}/msp
export CORE_PEER_MSPCONFIGPATH=${FABRIC_CFG_PATH}/crypto-config/peerOrganizations/${PEER_DOMAIN}/peers/${FAST_PEER_ADDRESS}.${PEER_DOMAIN}/msp


# export CORE_PEER_ID=${p_addr}
export CORE_PEER_ID=storage-peer
export CORE_PEER_ADDRESS=${p_addr}:7051
export CORE_PEER_GOSSIP_EXTERNALENDPOINT=${p_addr}:7051
export CORE_PEER_CHAINCODEADDRESS=${p_addr}:7052
export CORE_PEER_GOSSIP_USELEADERELECTION=false
export CORE_PEER_GOSSIP_ORGLEADER=false
export CORE_OPERATIONS_LISTENADDRESS=127.0.0.1:9443

# 使用专用数据目录避免与其他 peer 冲突
export CORE_PEER_FILESYSTEMPATH=/var/hyperledger/production/storage

# 使用专用数据目录避免与其他 peer 冲突
export CORE_PEER_FILESYSTEMPATH=/var/hyperledger/production/storage

# 创建专用数据目录
mkdir -p /var/hyperledger/production/storage
rm -rf /var/hyperledger/production/storage/* # clean up data from previous runs
(cd ${FABRIC_ROOT} && make peer)

# peer node start -s --storageAddr $p_addr:10000
peer node start  