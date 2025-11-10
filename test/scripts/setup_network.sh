#!/bin/bash

# Import variables and functions
# because these scripts are run in a separate docker container
source ../env.sh
source ./env.sh
source ./scripts/util.sh

function create_channel() {
  infoln "Checking organization crypto materials..."
  
  if [[ ! -d "./organization" ]]; then
    fatalln "Organization crypto materials directory not found: ./organization"
  fi

  # # 检查关键文件
  # local required_files=(
  #   "./organization/org1/user/org1admin@org1/msp/signcerts/cert.pem"
  #   "./organization/org1/user/org1admin@org1/msp/keystore"
  #   "./organization/org1/peer/org1peer0/msp/signcerts/cert.pem"
  #   "./organization/org0/orderer/org0orderer/msp/signcerts/cert.pem"
  # )

  # for file in "${required_files[@]}"; do
  #   if [[ ! -e "$file" ]]; then
  #     fatalln "Required file not found: $file"
  #   fi
  # done

  # infoln "All required crypto materials found. Creating channel..."
  ./scripts/create_channel.sh
}

function deploy_chaincode() {
  infoln "Deploying chaincode..."
  ./scripts/deploy_chaincode.sh
}

# # 验证 CONFIG_PATH 变量
# if [[ -z "${CONFIG_PATH}" ]]; then
#   fatalln "CONFIG_PATH variable not set"
# fi

export FABRIC_CFG_PATH="${CONFIG_PATH}"
infoln "FABRIC_CFG_PATH set to: ${FABRIC_CFG_PATH}"

# # 验证配置路径存在
# if [[ ! -d "${FABRIC_CFG_PATH}" ]]; then
#   fatalln "FABRIC_CFG_PATH directory not found: ${FABRIC_CFG_PATH}"
# fi

create_channel
deploy_chaincode