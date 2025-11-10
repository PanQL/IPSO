#!/bin/bash
source base_parameters.sh

p_addr=$(get_correct_peer_address ${FAST_PEER_ADDRESS})

# 设置配置文件路径为 core-endorser.yaml 所在目录
export FABRIC_CFG_PATH=${FABRIC_CFG_PATH}

export FABRIC_LOGGING_SPEC=INFO
# export CORE_PEER_MSPCONFIGPATH=${FABRIC_CFG_PATH}/crypto-config/peerOrganizations/${PEER_DOMAIN}/peers/$p_addr/msp
export CORE_PEER_MSPCONFIGPATH=${FABRIC_CFG_PATH}/crypto-config/peerOrganizations/${PEER_DOMAIN}/peers/${FAST_PEER_ADDRESS}.${PEER_DOMAIN}/msp

# export CORE_PEER_ID=${p_addr}
# export CORE_PEER_ADDRESS=${p_addr}:7051
# export CORE_PEER_GOSSIP_EXTERNALENDPOINT=${p_addr}:7051
# export CORE_PEER_CHAINCODEADDRESS=${p_addr}:7052
export CORE_PEER_GOSSIP_USELEADERELECTION=false
export CORE_PEER_GOSSIP_ORGLEADER=false
export CORE_PEER_ID=endorser-peer

# 使用专用数据目录避免与其他 peer 冲突
export CORE_PEER_FILESYSTEMPATH=/var/hyperledger/production/endorser

# 端口配置 - 这些将被 core-endorser.yaml 覆盖，仅用作备用
export CORE_PEER_LISTENADDRESS=0.0.0.0:7053
export CORE_PEER_ADDRESS=${p_addr}:7053
export CORE_PEER_GOSSIP_EXTERNALENDPOINT=${p_addr}:7053
export CORE_PEER_CHAINCODEADDRESS=${p_addr}:7054
export CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7054
export CORE_PEER_GOSSIP_BOOTSTRAP=${p_addr}:7051
export CORE_OPERATIONS_LISTENADDRESS=127.0.0.1:10443


echo "=== Endorser 配置验证 ==="
echo "FABRIC_CFG_PATH: $FABRIC_CFG_PATH"
echo "CORE_PEER_LISTENADDRESS: $CORE_PEER_LISTENADDRESS"
echo "CORE_PEER_CHAINCODELISTENADDRESS: $CORE_PEER_CHAINCODELISTENADDRESS"
echo "CORE_OPERATIONS_LISTENADDRESS: $CORE_OPERATIONS_LISTENADDRESS"
echo "CORE_PEER_FILESYSTEMPATH: $CORE_PEER_FILESYSTEMPATH"
echo "使用配置文件: ${FABRIC_CFG_PATH}/core-endorser.yaml"
echo "========================="

# 创建专用数据目录
mkdir -p /var/hyperledger/production/endorser
rm -rf /var/hyperledger/production/endorser/* # clean up data from previous runs
(cd ${FABRIC_ROOT} && make peer)
# peer node start -e --storageAddr $(get_correct_peer_address ${STORAGE_ADDRESS}):10000

# 启动 peer 并指定使用 core-endorser.yaml 配置文件
CORE_PEER_CFG=${FABRIC_CFG_PATH}/core-endorser.yaml peer node start 
