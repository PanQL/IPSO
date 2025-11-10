#!/bin/bash

source ${SCRIPT_PATH}/util.sh

function enroll_ca_admin() {
  local ca_admin_name="$1"
  local ca_admin_password="$2"

  infoln "Enrolling the CA admin"

  set -x
  fabric-ca-client enroll -u https://${ca_admin_name}:${ca_admin_password}@${CA_HOST}:${CA_PORT} --caname "${CA_NAME}" --tls.certfiles "${CA_TLS_CERT}"
  res=$?
  { set +x; } 2>/dev/null

  echo "NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/${CA_HOST}-${CA_PORT}-${CA_NAME}.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/${CA_HOST}-${CA_PORT}-${CA_NAME}.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/${CA_HOST}-${CA_PORT}-${CA_NAME}.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/${CA_HOST}-${CA_PORT}-${CA_NAME}.pem
    OrganizationalUnitIdentifier: orderer" >"${ORG_PATH}/msp/config.yaml"

  verify_result $res 'Enroll the CA admin' 'Failed to enroll the CA admin'
}

function register_enroll_peer() {
  local peer_name="$1"
  local peer_password="$2"
  local peer_fullname="${ORG_NAME}${peer_name}"
  local peer_path="${ORG_PATH}/peer/${peer_fullname}"

  mkdir -p ${peer_path}

  infoln "Registering ${peer_name}"
  set -x
  fabric-ca-client register --caname ${CA_NAME} --id.name ${peer_name} --id.secret ${peer_password} --id.type peer --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Register ${peer_name}" "Failed to register ${peer_name}"

  infoln "Enrolling ${peer_name}"
  set -x
  fabric-ca-client enroll -u https://${peer_name}:${peer_password}@${CA_HOST}:${CA_PORT} --caname ${CA_NAME} -M ${peer_path}/msp --csr.hosts ${peer_fullname} --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Enroll ${peer_name}" "Failed to enroll ${peer_name}"

  cp ${ORG_PATH}/msp/config.yaml ${peer_path}/msp/config.yaml

  infoln "Generating TLS certificates for ${peer_name} "
  set -x
  fabric-ca-client enroll -u https://${peer_name}:${peer_password}@${CA_HOST}:${CA_PORT} --caname ${CA_NAME} -M ${peer_path}/tls --enrollment.profile tls --csr.hosts ${peer_fullname} --csr.hosts ${CA_HOST} --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Generate TLS certificates for ${peer_name}" "Failed to generate TLS certificates for ${peer_name}"

  cp ${peer_path}/tls/tlscacerts/* ${peer_path}/tls/ca.crt
  cp ${peer_path}/tls/signcerts/* ${peer_path}/tls/server.crt
  cp ${peer_path}/tls/keystore/* ${peer_path}/tls/server.key

  mkdir -p ${ORG_PATH}/msp/tlscacerts
  cp ${peer_path}/tls/tlscacerts/* ${ORG_PATH}/msp/tlscacerts/ca.crt

  mkdir -p ${ORG_PATH}/tlsca
  cp ${peer_path}/tls/tlscacerts/* ${ORG_PATH}/tlsca/tlsca.${ORG_NAME}-cert.pem

  mkdir -p ${ORG_PATH}/ca
  cp ${peer_path}/msp/cacerts/* ${ORG_PATH}/ca/ca.${ORG_NAME}-cert.pem

  mv ${peer_path}/msp/keystore/* ${peer_path}/msp/keystore/key.pem &>/dev/null
  mv ${peer_path}/msp/signcerts/* ${peer_path}/msp/signcerts/cert.pem &>/dev/null
}

function register_enroll_user() {
  local user_name="$1"
  local user_password="$2"
  local user_fullname="${user_name}@${ORG_NAME}"
  local user_path="${PWD}/organization/${ORG_NAME}/user/${user_fullname}"

  mkdir -p ${user_path}

  infoln "Registering user '${user_name}'"
  set -x
  fabric-ca-client register --caname ${CA_NAME} --id.name ${user_name} --id.secret ${user_password} --id.type client --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Register user '${user_name}'" "Failed to register user '${user_name}'"

  infoln "Enrolling user '${user_name}'"
  set -x
  fabric-ca-client enroll -u https://${user_name}:${user_password}@${CA_HOST}:${CA_PORT} --caname ${CA_NAME} -M ${user_path}/msp --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Enroll user '${user_name}'" "Failed to enroll user '${user_name}'"

  cp ${ORG_PATH}/msp/config.yaml ${user_path}/msp/config.yaml

  mv ${user_path}/msp/keystore/* ${user_path}/msp/keystore/key.pem &>/dev/null
  mv ${user_path}/msp/signcerts/* ${user_path}/msp/signcerts/cert.pem &>/dev/null
}

function register_enroll_orderer() {
  local orderer_name="$1"
  local orderer_password="$2"
  local orderer_fullname="${ORG_NAME}${orderer_name}"
  local orderer_path="${PWD}/organization/${ORG_NAME}/orderer/${orderer_fullname}"

  mkdir -p ${orderer_path}

  infoln "Registering orderer '${orderer_name}'"
  set -x
  fabric-ca-client register --caname ${CA_NAME} --id.name ${orderer_name} --id.secret ${orderer_password} --id.type orderer --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Register orderer '${orderer_name}'" "Failed to register orderer '${orderer_name}'"

  infoln "Enrolling orderer ${orderer_name}"
  set -x
  fabric-ca-client enroll -u https://${orderer_name}:${orderer_password}@${CA_HOST}:${CA_PORT} --caname ${CA_NAME} -M ${orderer_path}/msp --csr.hosts ${orderer_fullname} --csr.hosts localhost --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Enroll orderer '${orderer_name}'" "Failed to enroll orderer '${orderer_name}'"

  cp ${ORG_PATH}/msp/config.yaml ${orderer_path}/msp/config.yaml

  infoln "Generating TLS certificates for '${orderer_name}'"
  set -x
  fabric-ca-client enroll -u https://${orderer_name}:${orderer_password}@${CA_HOST}:${CA_PORT} --caname ${CA_NAME} -M ${orderer_path}/tls --enrollment.profile tls --csr.hosts ${orderer_fullname} --csr.hosts ${CA_HOST} --csr.hosts localhost --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Generate TLS certificates for '${orderer_name}'" "Failed to generate TLS certificates for '${orderer_name}'"

  cp ${orderer_path}/tls/tlscacerts/* ${orderer_path}/tls/ca.crt
  cp ${orderer_path}/tls/signcerts/* ${orderer_path}/tls/server.crt
  cp ${orderer_path}/tls/keystore/* ${orderer_path}/tls/server.key

  mkdir -p ${ORG_PATH}/msp/tlscacerts
  cp ${orderer_path}/tls/tlscacerts/* ${ORG_PATH}/msp/tlscacerts/ca.crt

  mkdir -p ${ORG_PATH}/tlsca
  cp ${orderer_path}/tls/tlscacerts/* ${ORG_PATH}/tlsca/tlsca.${ORG_NAME}-cert.pem

  mkdir -p ${ORG_PATH}/ca
  cp ${orderer_path}/msp/cacerts/* ${ORG_PATH}/ca/ca.${ORG_NAME}-cert.pem

  mv ${orderer_path}/msp/keystore/* ${orderer_path}/msp/keystore/key.pem &>/dev/null
  mv ${orderer_path}/msp/signcerts/* ${orderer_path}/msp/signcerts/cert.pem &>/dev/null
}

function register_enroll_org_admin() {
  local admin_name="$1"
  local admin_password="$2"
  local admin_folder_name="$3"
  local admin_fullname="${admin_name}@${ORG_NAME}"
  local admin_path="${PWD}/organization/${ORG_NAME}/user/${admin_fullname}"

  infoln "Registering org admin '${admin_name}'"
  set -x
  fabric-ca-client register --caname ${CA_NAME} --id.name ${admin_name} --id.secret ${admin_password} --id.type admin --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Register org admin '${admin_name}'" "Failed to register org admin '${admin_name}'"

  mkdir -p ${admin_path}

  infoln "Enrolling org admin '${admin_name}'"
  set -x
  fabric-ca-client enroll -u https://${admin_name}:${admin_password}@${CA_HOST}:${CA_PORT} --caname ${CA_NAME} -M ${admin_path}/msp --tls.certfiles ${CA_TLS_CERT}
  res=$?
  { set +x; } 2>/dev/null
  verify_result $res "Enroll org admin '${admin_name}'" "Failed to enroll org admin '${admin_name}'"

  cp ${ORG_PATH}/msp/config.yaml ${admin_path}/msp/config.yaml

  mv ${admin_path}/msp/keystore/* ${admin_path}/msp/keystore/key.pem &>/dev/null
  mv ${admin_path}/msp/signcerts/* ${admin_path}/msp/signcerts/cert.pem &>/dev/null
}

function 1p1o_create_org1() {
  export ORG_NAME='org1'
  export ORG_PATH="${PWD}/organization/${ORG_NAME}"
  export CA_NAME='ca-org1'
  export CA_HOST='localhost'
  export CA_PORT='7054'
  export CA_PATH="${PWD}/ca/${ORG_NAME}"
  export CA_TLS_CERT="${CA_PATH}/tls-cert.pem"

  mkdir -p ${ORG_PATH}

  export FABRIC_CA_CLIENT_HOME=${ORG_PATH}

  enroll_ca_admin 'admin' 'adminpw'

  register_enroll_peer 'peer0' 'peer0pw'

  register_enroll_user 'user1' 'user1pw'

  register_enroll_org_admin 'org1admin' 'org1adminpw'
}

function 4p1o_create_org1() {
  export ORG_NAME='org1'
  export ORG_PATH="${PWD}/organization/${ORG_NAME}"
  export CA_NAME='ca-org1'
  export CA_HOST='localhost'
  export CA_PORT='7054'
  export CA_PATH="${PWD}/ca/${ORG_NAME}"
  export CA_TLS_CERT="${CA_PATH}/tls-cert.pem"

  mkdir -p ${ORG_PATH}

  export FABRIC_CA_CLIENT_HOME=${ORG_PATH}

  enroll_ca_admin 'admin' 'adminpw'

  register_enroll_peer 'peer0' 'peer0pw'

  register_enroll_peer 'peer1' 'peer1pw'

  register_enroll_user 'user1' 'user1pw'

  register_enroll_org_admin 'org1admin' 'org1adminpw'
}

function 4p1o_create_org2() {
  export ORG_NAME='org2'
  export ORG_PATH="${PWD}/organization/${ORG_NAME}"
  export CA_NAME='ca-org2'
  export CA_HOST='localhost'
  export CA_PORT='8054'
  export CA_PATH="${PWD}/ca/${ORG_NAME}"
  export CA_TLS_CERT="${CA_PATH}/tls-cert.pem"

  mkdir -p ${ORG_PATH}

  export FABRIC_CA_CLIENT_HOME=${ORG_PATH}

  enroll_ca_admin 'admin' 'adminpw'

  register_enroll_peer 'peer0' 'peer0pw'

  register_enroll_peer 'peer1' 'peer1pw'

  register_enroll_user 'user1' 'user1pw'

  register_enroll_org_admin 'org2admin' 'org2adminpw'
}

function create_org0() {
  export ORG_NAME='org0'
  export ORG_PATH="${PWD}/organization/${ORG_NAME}"
  export CA_NAME='ca-org0'
  export CA_HOST='localhost'
  export CA_PORT='9054'
  export CA_PATH="${PWD}/ca/${ORG_NAME}"
  export CA_TLS_CERT="${CA_PATH}/tls-cert.pem"

  mkdir -p ${ORG_PATH}

  export FABRIC_CA_CLIENT_HOME=${ORG_PATH}

  enroll_ca_admin 'admin' 'adminpw'

  register_enroll_org_admin 'ordererAdmin' 'ordererAdminpw'

  register_enroll_orderer 'orderer' 'ordererpw'
}

if [[ ${DEPLOY_SCHEME} == 1p1o ]]; then
  infoln "Create org1 Identities"
  1p1o_create_org1

  infoln "Create org0 Identities"
  create_org0

elif [[ ${DEPLOY_SCHEME} == 4p1o ]]; then
  infoln "Create org1 Identities"
  4p1o_create_org1

  infoln "Create org2 Identities"
  4p1o_create_org2

  infoln "Create org0 Identities"
  create_org0
fi
