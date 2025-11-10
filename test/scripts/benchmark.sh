#!/bin/sh

# Import variables
source ./env.sh

function run_e2e() {
  # Creating accounts
  set -x
  cp ${TAPE_CONFIG_FILE_PUT} result/tape-put.yaml
  tape --config ${TAPE_CONFIG_FILE_PUT} >${TAPE_STDOUT_FILE_PUT} 2>${TAPE_STDERR_FILE_PUT}
  cp ACCOUNTS.txt result/ACCOUNTS_PUT.txt
  cp TRANSACTIONS.txt result/TRANSACTIONS_PUT.txt
  cp success_txns.txt result/success_txns_put.txt


  yq -i ".rate = ${TAPE_GENERATION_RATE}" ${TAPE_CONFIG_FILE_CONFLICT} # Set the rate to 1000
  yq -i ".connNum = ${TAPE_CONN_NUM}" ${TAPE_CONFIG_FILE_CONFLICT} # Set the connection number
  yq -i ".conflictRatio= ${TAPE_CR}" ${TAPE_CONFIG_FILE_CONFLICT} # Set the conflict ratio

  # Transfering money
  cp ${TAPE_CONFIG_FILE_CONFLICT} result/tape-conflict.yaml
  tape --config ${TAPE_CONFIG_FILE_CONFLICT} >${TAPE_STDOUT_FILE_CONFLICT} 2>${TAPE_STDERR_FILE_CONFLICT}
  cp ACCOUNTS.txt result/ACCOUNTS_CONFLICT.txt
  cp TRANSACTIONS.txt result/TRANSACTIONS_CONFLICT.txt
  cp success_txns.txt result/success_txns_conflict.txt
  { set +x; } 2>/dev/null
}

run_e2e
