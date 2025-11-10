#!/bin/bash

source ./consym.sh

CUR_DIR=$(pwd)
RESULT_DIR="${CUR_DIR}/$(date +%Y%m%d_%H%M%S)_result"
mkdir -p "$RESULT_DIR"

export TAPE_CONN_NUM=16
for hlf_typ in "sharp"; do
  cd ${CUR_DIR}
  export HLF_TYPE=${hlf_typ}
  source ./env.sh
  echo "Running with HLF_TYPE: ${HLF_TYPE}"
  # for cr in 0.1 0.5 0.9; do
  for cr in 0.5; do
    export TAPE_CR=${cr}
    for rate in 1000; do
    # for rate in 500 1000 2000 2500 3000 3200 3400 3600 4000 5000; do
    # for rate in 1000 2000 3000; do
      export TAPE_GENERATION_RATE=${rate}
      echo "Running with rate: ${TAPE_GENERATION_RATE}, cr: ${cr}"
      clean
      run
      mv ${CUR_DIR}/test/result ${RESULT_DIR}/${HLF_TYPE}_conn${TAPE_CONN_NUM}_${TAPE_GENERATION_RATE}_cr${cr}
    done
  done
done
# for rate in 500 1000 2000 4000 8000 12000 16000 20000 0; do
# for rate in 100 500 1000 5000 10000 15000 20000 25000 30000 40000 50000 0; do

# export TAPE_GENERATION_RATE=0
# for conn_num in 1 2 4 8 12 16 24 32; do
#   export TAPE_CONN_NUM=${conn_num}
#   echo "Running with connNum: ${TAPE_CONN_NUM}"
#   clean
#   run
#   mv ${CUR_DIR}/test/result ${RESULT_DIR}/${HLF_TYPE}_${TAPE_GENERATION_RATE}_conn${TAPE_CONN_NUM}
# done