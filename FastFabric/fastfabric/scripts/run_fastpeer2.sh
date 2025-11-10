#!/bin/bash
source base_parameters.sh

export FABRIC_LOGGING_SPEC=INFO
export CORE_PEER_MSPCONFIGPATH=${FABRIC_CFG_PATH}/crypto-config/peerOrganizations/${PEER_DOMAIN}/peers/${FAST_PEER_ADDRESS}.${PEER_DOMAIN}/msp

p_addr=$(get_correct_peer_address $FAST_PEER_ADDRESS)
# export CORE_PEER_ID=${p_addr}
export CORE_PEER_ID=fast-peer


# FastPeer 使用独立端口避免冲突
export CORE_PEER_LISTENADDRESS=0.0.0.0:7055
export CORE_PEER_ADDRESS=${p_addr}:7055
export CORE_PEER_GOSSIP_EXTERNALENDPOINT=${p_addr}:7055
export CORE_PEER_CHAINCODEADDRESS=${p_addr}:7056
export CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7056
export CORE_PEER_GOSSIP_BOOTSTRAP=${p_addr}:7051
export CORE_OPERATIONS_LISTENADDRESS=127.0.0.1:10444
export CORE_PEER_GOSSIP_USELEADERELECTION=false
export CORE_PEER_GOSSIP_ORGLEADER=true

# 使用专用数据目录避免与其他 peer 冲突
export CORE_PEER_FILESYSTEMPATH=/var/hyperledger/production/fastpeer

# 使用专用数据目录避免与其他 peer 冲突
export CORE_PEER_FILESYSTEMPATH=/var/hyperledger/production/fastpeer

echo "=== FastPeer 配置验证 ==="
echo "CORE_PEER_LISTENADDRESS: $CORE_PEER_LISTENADDRESS"
echo "CORE_PEER_CHAINCODELISTENADDRESS: $CORE_PEER_CHAINCODELISTENADDRESS"
echo "CORE_OPERATIONS_LISTENADDRESS: $CORE_OPERATIONS_LISTENADDRESS"
echo "CORE_PEER_FILESYSTEMPATH: $CORE_PEER_FILESYSTEMPATH"
echo "========================="

# 创建专用数据目录
mkdir -p /var/hyperledger/production/fastpeer
rm -rf /var/hyperledger/production/fastpeer/* # clean up data from previous runs
(cd ${FABRIC_ROOT} && make peer)

# peer node start can be run without the storageAddr. In that case those modules will not be decoupled to different nodes
s=""
if [[ ! -z ${STORAGE_ADDRESS} ]]
then
    echo "Starting with decoupled storage server ${STORAGE_ADDRESS}"
    s="--storageAddr $(get_correct_peer_address ${STORAGE_ADDRESS}):10000"
fi

peer node start ${s}

