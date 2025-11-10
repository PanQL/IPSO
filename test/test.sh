#!/bin/bash

####################
# Import variables
####################
source ../env.sh
source ./env.sh

####################
# Import functions
####################
source ${SCRIPT_PATH}/util.sh

####################
# Functions
####################
function prepare_artifact() {
  ${SCRIPT_PATH}/prepare_artifact.sh
}

function deploy_service() {
  mkdir -p ${RESULT_PATH}

  export STATEDATABASE='goleveldb'
  if [[ ${HLF_TYPE} == 'sharp' ]]; then
    export STATEDATABASE='UstoreDB' # FabricSharp requires UstoreDB to use the occ-sharp scheduler
  fi

  if [[ ${DEPLOY_MODE} == 'swarm' ]]; then
    # infoln "Initialize swarm with '${INIT_NODE_IP}' as manager node"
    # sudo -E docker swarm init --advertise-addr ${INIT_NODE_IP} &>/dev/null

    infoln "Create overlay network '${NETWORK_NAME}'"
    sudo -E docker network create --driver overlay --attachable ${NETWORK_NAME}

    infoln "Deploy service to swarm"
    sudo -E docker stack deploy --detach=false -c ${HLF_SWARM_FILE} --resolve-image never ${PROJECT_NAME}
  fi

  if [[ ${DEPLOY_MODE} == 'compose' ]]; then
    infoln "Deploy service to compose"
    sudo -E docker compose -f ${HLF_COMPOSE_FILE} -p ${PROJECT_NAME} up -d 2>&1
  fi
}

function setup_network() {
  infoln "Creating channel and deploy network"
  # sudo docker logs org0orderer
  # # 检查部署状态
  # sleep 10  # 给容器一些启动时间
  
  # infoln "=== Debugging container startup ==="
  # infoln "All containers:"
  # sudo docker ps -a
  
  # infoln "All services (if swarm):"
  # sudo docker service ls 2>/dev/null || echo "Not using swarm mode"
  
  # infoln "Looking for cli container..."
  # sudo docker ps | grep cli || echo "No cli container running"
  
  # local cli_id=''
  # local attempt=0

  while :; do
    infoln "Waiting for cli to start up..."
    cli_id=$(sudo -E docker container ls -f 'name=cli' -q 2>/dev/null)
    if [[ -n ${cli_id} ]]; then
      break
    fi
    sleep ${DELAY}
  done

  sudo -E docker exec ${cli_id} bash -c "./scripts/setup_network.sh"
}

function benchmark() {
  infoln "Using tape to benchmark the network"
  local tape_id=''
  while :; do
    infoln "Waiting for tape to start up..."
    tape_id=$(sudo -E docker container ls -f 'name=tape' -q 2>/dev/null)
    if [[ -n ${tape_id} ]]; then
      break
    fi
    sleep ${DELAY}
  done

  sudo -E docker exec ${tape_id} sh "./scripts/benchmark.sh"
}

function leave() {
  if [[ ${DEPLOY_MODE} == 'swarm' ]]; then
    infoln "Remove services from swarm"
    sudo -E docker stack rm ${PROJECT_NAME}

    infoln "Removing chaincode containers"
    CONTAINER_IDS=$(sudo -E docker ps -a | awk '($2 ~ /dev-.*/) {print $1}')
    if [[ -z "${CONTAINER_IDS}" || "${CONTAINER_IDS}" == " " ]]; then
      infoln "No containers available for deletion"
    else
      sudo -E docker container rm -f ${CONTAINER_IDS}
    fi

    infoln "Removing chaincode images"
    DOCKER_IMAGE_IDS=$(sudo -E docker images | awk '($1 ~ /dev-.*/) {print $3}')
    if [[ -z "${DOCKER_IMAGE_IDS}" || "${DOCKER_IMAGE_IDS}" == " " ]]; then
      infoln "No images available for deletion"
    else
      sudo -E docker image remove -f ${DOCKER_IMAGE_IDS}
    fi

    sleep 10
    infoln "Remove overlay network '${NETWORK_NAME}'"
    sudo -E docker network rm ${NETWORK_NAME}

    # infoln "Manager '${INIT_NODE_IP}' leaves swarm"
    # sudo -E docker swarm leave --force
  fi

  if [[ ${DEPLOY_MODE} == 'compose' ]]; then
    infoln "Remove services"
    sudo -E docker compose -f ${HLF_COMPOSE_FILE} down --volumes --remove-orphans

    infoln "Remove chaincode containers"
    CONTAINER_IDS=$(sudo -E docker ps -a | awk '($2 ~ /dev-.*/) {print $1}')
    if [[ -z "${CONTAINER_IDS}" || "${CONTAINER_IDS}" == " " ]]; then
      infoln "No containers available for deletion"
    else
      sudo -E docker container rm -f ${CONTAINER_IDS}
    fi

    infoln "Remove chaincode images"
    DOCKER_IMAGE_IDS=$(sudo -E docker images | awk '($1 ~ /dev-.*/) {print $3}')
    if [[ -z "${DOCKER_IMAGE_IDS}" || "${DOCKER_IMAGE_IDS}" == " " ]]; then
      infoln "No images available for deletion"
    else
      sudo -E docker image rm -f ${DOCKER_IMAGE_IDS}
    fi
  fi
}


CMD=$1
if [[ $CMD == run ]]; then
  # replace the target_chaincode.tar.gz with the one built for the current HLF_TYPE
  # rm -f ${PWD}/chaincode/build/target_chaincode.tar.gz
  # cp ${PWD}/chaincode/build/${HLF_TYPE}_smallbank.tar.gz ${PWD}/chaincode/build/target_chaincode.tar.gz

  prepare_artifact
  if [[ ${DEPLOY_MODE} == 'swarm' ]]; then
    infoln "Distribute test scripts to all nodes"
    for host in ${HOSTS[@]}; do
      infoln "The host is '$host'"
      # Skip the current host
      ip a | grep "$host" >/dev/null
      if [[ $? == 0 ]]; then
        continue
      fi

      target="$USER@$host"
      target_path="/home/$USER/consym"

      # Copy the image tar files to the target host's path
      infoln "Cleaning old test scripts for '$host'"
      set -x
      ssh ${target} "cd ${target_path} && rm -rf test"
      set +x
      # scp ${PEER_IMAGE_FILE} ${ORDERER_IMAGE_FILE} ${TOOLS_IMAGE_FILE} ${TAPE_IMAGE_FILE} ${target}:${target_path}

      infoln "Distributing new test scripts to '$host'"
      set -x
      sudo -E scp -r ${target_path}/test ${target}:${target_path} > /dev/null
      # ssh ${target} "cd ${target_path} && docker load --input ${PEER_IMAGE_FILE_NAME} && docker load --input ${ORDERER_IMAGE_FILE_NAME} && docker load --input ${TOOLS_IMAGE_FILE_NAME} && docker load --input ${TAPE_IMAGE_FILE_NAME}"
      set +x
    done
  fi
  # set -x
  # ssh panda@10.213.3.13 "cd ~/consym && rm -rf test"
  # scp -r ~/consym/test panda@10.213.3.13:~/consym
  # set +x
  deploy_service
  setup_network
  # export bench_begin=$(date +'%Y-%m-%d %H:%M:%S')
  sudo -E tc qdisc add dev eno1 root netem delay 100ms 10ms
  benchmark
  sudo -E tc qdisc del dev eno1 root netem
  # sudo docker service logs hlf_org1peer0 &>hlf_org1peer0.log
  # sudo docker service logs hlf_org0orderer &>hlf_org0orderer.log
  # sudo -E journalctl -o short-iso SYSLOG_IDENTIFIER=org1peer0 --since "${bench_begin}" > /home/panda/org1peer0.log
  # docker service logs hlf_org0orderer > /home/panda/consym/test/result/${HLF_TYPE}_org0orderer.log 2>&1
  # docker service logs hlf_org1peer0 > /home/panda/consym/test/result/${HLF_TYPE}_org1peer0.log 2>&1
  # docker logs org0orderer > /home/panda/consym/test/result/${HLF_TYPE}_org0orderer.log 2>&1
  # docker logs org1peer0 > /home/panda/consym/test/result/${HLF_TYPE}_org1peer0.log 2>&1
  leave
elif [[ $CMD == analyze ]]; then
  analyze
fi
