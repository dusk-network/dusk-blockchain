#!/bin/bash

SCRIPT=$(basename ${BASH_SOURCE[0]})
QTD=2 #QTD=2 == 3 nodes since it counts from zero
TYPE=branch
DUSK_VERSION=v0.3.0
INIT=true
MOCK=true
VOUCHER=true
FILEBEAT=false
MOCK_ADDRESS=127.0.0.1:9191
currentDir=$(pwd)

# define command exec location
duskCMD="../bin/dusk"

usage() {
  echo "Usage: $SCRIPT"
  echo "Optional command line arguments"
  echo "-c <number>  -- TYPE to use. eg: release"
  echo "-q <number>  -- QTD of validators to run. eg: 5"
  echo "-s <number>  -- DUSK_VERSION version. eg: v0.3.0"
  echo "-i <number>  -- INIT instances (true or false). eg: true"
  echo "-m <number>  -- MOCK instances (true or false). eg: true"
  echo "-f <number>  -- FILEBEAT instances (true or false). eg: true"

  exit 1
}

while getopts "h?c:q:s:i:m:f:" args; do
case $args in
    h|\?)
      usage;
      exit;;
    c ) TYPE=${OPTARG};;
    q ) QTD=${OPTARG};;
    s ) DUSK_VERSION=${OPTARG};;
    i ) INIT=${OPTARG};;
    m ) MOCK=${OPTARG};;
    f ) FILEBEAT=${OPTARG};;
  esac
done

set -euxo pipefail

if [ "${TYPE}" == "branch" ]; then
  GOBIN=$(pwd)/bin go run scripts/build.go install
fi

#init dusk
init_dusk_func() {
  echo "init Dusk node $i ..."
  DDIR="${currentDir}/devnet/dusk_data/dusk${i}"
  LOGS="${currentDir}/devnet/dusk_data/logs"

  # cleanup
  rm -rf "${DDIR}"
  rm -rf "${LOGS}"
  rm -rf "${DDIR}/walletDB/"

  mkdir -p "${DDIR}"
  mkdir -p "${LOGS}"
  mkdir -p "${DDIR}/walletDB/"

  cat <<EOF > "${DDIR}"/dusk.toml
[consensus]
  defaultamount = 50
  defaultlocktime = 1000
  defaultoffset = 10

[database]
  dir = "${DDIR}/chain/"
  driver = "heavy_v0.1.0"

[general]
  network = "testnet"
  walletonly = "false"

[genesis]
  legacy = true

[gql]
  address = "127.0.0.1:$((9500+$i))"
  enabled = "true"
  network = "tcp"

[logger]
  output = "${DDIR}/dusk"
  level = "trace"
  format = "json"

[mempool]
  maxinvitems = "10000"
  maxsizemb = "100"
  pooltype = "hashmap"
  prealloctxs = "100"

[network]
  port = "$((7100+$i))"

  [network.seeder]
    addresses = ["127.0.0.1:8081"]

[rpc]
  address = "${DDIR}/dusk-grpc.sock"
  enabled = "true"
  network = "unix"

  [rpc.rusk]
    address = "127.0.0.1:$((10000+$i))"
    contracttimeout = 6000
    defaulttimeout = 1000
    network = "tcp"

[wallet]
  file = "${currentDir}/harness/data/wallet-$((9000+$i)).dat"
  store = "${DDIR}/walletDB/"
EOF

  echo "init Dusk node $i, done."
}

#init dusk
init_filebeat_func() {
  echo "init FILEBEAT node $i ..."
  DDIR="${currentDir}/devnet/dusk_data/dusk${i}"
  rm -rf "${DDIR}"/filebeat

  cat <<EOF > "${DDIR}"/filebeat.json.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - ${DDIR}/dusk$((7100+$i)).log
  exclude_files: ['\.gz$']
  encoding: plain

  json.message_key: msg
  json.message_level: level
  json.message_category: category
  json.message_process: category
  json.message_topic: topic

  json.message_recipients: recipients
  json.message_step: step
  json.message_round: round
  json.message_id: id
  json.message_sender: sender
  json.message_agreement: agreement
  json.message_quorum: quorum

  json.message_new_best: new_best

  json.message_block_hash: block_hash
  json.message_prefix: prefix
  json.message_hash: hash

  json.keys_under_root: true
name: dusk$((7100+$i))
output.logstash:
  hosts: ["0.0.0.0:5000"]
  #index: "devnet-v1"
  tags: ["devnet","dusk${i}"]
  #ignore_older: "5h"
  #close_inactive: "4h"
  #close_renamed: true
  #clean_inactive: true
  #clean_removed: true
  #tail_files: true
  encoding: plain
fields:
  env: dusk$((7100+$i))
#  type: node
#processors:
#  - add_host_metadata: ~
#  - add_cloud_metadata: ~
EOF

  chmod go-w "${DDIR}"/filebeat.json.yml

  echo "init FILEBEAT node $i, done."
}

# start dusk mocks
start_dusk_mock_rusk_func() {
  echo "starting Dusk Rusk mock $i ..."

  DDIR="${currentDir}/devnet/dusk_data/dusk${i}"
  WALLET_FILE="${currentDir}/harness/data/wallet-$((9000+$i)).dat"

  CMD="${currentDir}/bin/utils mockrusk --rusknetwork tcp --ruskaddress 127.0.0.1:$((10000+$i)) --walletstore ${DDIR}/walletDB/ --walletfile ${WALLET_FILE}"
  ${CMD} >> "${currentDir}/devnet/dusk_data/logs/mock$i.log" 2>&1 &

  EXEC_PID=$!
  echo "started Dusk Rusk Mock node $i, pid=$EXEC_PID"
}

# start dusk
start_dusk_func() {
  echo "starting Dusk node $i ..."

  DDIR="${currentDir}/devnet/dusk_data/dusk${i}"

  CMD="${currentDir}/bin/dusk --config ${DDIR}/dusk.toml"
  ${CMD} >> "${currentDir}/devnet/dusk_data/logs/dusk$i.log" 2>&1 &

  EXEC_PID=$!
  echo "started Dusk node $i, pid=$EXEC_PID"
  sleep 1

  # load wallet cmd
  LOADWALLET_CMD="./bin/utils walletutils --grpchost unix://${DDIR}/dusk-grpc.sock --walletcmd loadwallet --walletpassword password"
  ${LOADWALLET_CMD} >> "${currentDir}/devnet/dusk_data/logs/load_wallet$i.log" 2>&1 &

}

start_dusk_mock_func() {
  echo "starting Dusk Utils mock ..."

  CMD="${currentDir}/bin/utils mock --grpcmockhost $MOCK_ADDRESS"
  ${CMD} >> "${currentDir}/devnet/dusk_data/logs/mock.log" 2>&1 &

  EXEC_PID=$!
  echo "started Dusk Utils mock, pid=$EXEC_PID"
}

start_voucher_func() {
  echo "starting Dusk Voucher ..."

  CMD="${currentDir}/bin/voucher"
  ${CMD} >> "${currentDir}/devnet/dusk_data/logs/voucher.log" 2>&1 &

  EXEC_PID=$!
  echo "started Dusk Voucher, pid=$EXEC_PID"
}

start_filebeat_func() {
  echo "starting FILEBEAT node $i ..."

  DDIR="${currentDir}/devnet/dusk_data/dusk${i}"
  filebeat -e -path.config "${DDIR}" -path.data "${DDIR}/filebeat" -c filebeat.json.yml -d "*" >> "${currentDir}/devnet/dusk_data/logs/filebeat${i}.log" 2>&1 &

  EXEC_PID=$!
  echo "started FILEBEAT, pid=$EXEC_PID"
}

if [ "${INIT}" == "true" ]; then

  for i in $(seq 0 "$QTD"); do
    init_dusk_func "$i"
  done

fi

sleep 1

if [ "${VOUCHER}" == "true" ]; then
  start_voucher_func
fi

sleep 1

if [ "${MOCK}" == "true" ]; then
  start_dusk_mock_func

  for i in $(seq 0 "$QTD"); do
    start_dusk_mock_rusk_func "$i"
  done
fi

if [ "${FILEBEAT}" == "true" ]; then

  for i in $(seq 0 "$QTD"); do
    init_filebeat_func "$i"
    start_filebeat_func "$i"
  done

fi

sleep 1

for i in $(seq 0 "$QTD"); do
  start_dusk_func "$i"
done

echo "done starting Dusk stack"
exit 0