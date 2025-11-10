#!/bin/bash

####################
# Configuration
####################

# for subcommand './consym.sh build'
export BUILD_HLF_PEER_IMAGE='true'
export BUILD_HLF_ORDERER_IMAGE='true'
export BUILD_HLF_TOOLS_IMAGE='true'
export BUILD_TAPE_IMAGE='false'

# for subcommand './consym.sh clean'
export CLEAN_HLF_PEER_IMAGE='false'
export CLEAN_HLF_ORDERER_IMAGE='false'
export CLEAN_HLF_TOOLS_IMAGE='false'
export CLEAN_TAPE_IMAGE='false'
export CLEAN_DANGLING_IMAGE='true'
export CLEAN_RESULT='true'

export HLF_TYPE='vanilla'        # 'vanilla' 'fast' 'sharp' 'consym' 'strawman'
export HLF_IMAGE_PREFIX="fabric-${HLF_TYPE}"
# if [[ ${HLF_TYPE} == 'strawman' ]]; then
#   export HLF_IMAGE_PREFIX="fabric-consym"
# fi
export HLF_OLD_TAG='latest'
export HLF_NEW_TAG="${HLF_TYPE}"

export HOSTS=(
  10.213.3.11
  10.213.3.12
  10.213.3.13
  10.30.6.1
)
export INIT_NODE_IP=${HOSTS[0]}

####################
# Path
####################
export CONSYM_PATH="${PWD}"
export TAPE_PATH="${CONSYM_PATH}/tape"
export SHARP_PATH="${CONSYM_PATH}/FabricSharp"
export TEST_PATH="${CONSYM_PATH}/test"
export IMAGE_PATH="${CONSYM_PATH}/image"
export PATH="${TEST_PATH}/bin:${PATH}"
if [[ ${HLF_TYPE} == vanilla ]]; then
  export HLF_PATH="${CONSYM_PATH}/VanillaHLF"
elif [[ ${HLF_TYPE} == fast ]]; then
  export HLF_PATH="${CONSYM_PATH}/FastFabric"
elif [[ ${HLF_TYPE} == sharp ]]; then
  export HLF_PATH="${CONSYM_PATH}/FabricSharp"
elif [[ ${HLF_TYPE} == strawman ]]; then
  export HLF_PATH="${CONSYM_PATH}/ConsymHLF"
else
  export HLF_PATH="${CONSYM_PATH}/ConsymHLF"
fi
