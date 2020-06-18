#!/bin/bash

SCRIPT=$(basename ${BASH_SOURCE[0]})
QTD=2 #QTD=2 == 3 nodes since it counts from zero
TYPE=branch
DUSK_VERSION=v0.3.0
INIT=true
MOCK=true

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

  exit 1
}

while getopts "h?c:q:s:i:m:" args; do
case $args in
    h|\?)
      usage;
      exit;;
    c ) TYPE=${OPTARG};;
    q ) QTD=${OPTARG};;
    s ) DUSK_VERSION=${OPTARG};;
    i ) INIT=${OPTARG};;
    m ) MOCK=${OPTARG};;
  esac
done

set -euxo pipefail

if [ "${TYPE}" == "branch" ]; then
  GOBIN=$(pwd)/bin go run scripts/build.go install
fi

#init dusk
init_dusk_func() {
  echo "init Dusk node $i ..."
  currentDir=$(pwd)
  DDIR="${currentDir}/devnet/dusk_data/dusk${i}"
  LOGS="${currentDir}/devnet/dusk_data/logs"

  # cleanup
  rm -rf "${DDIR}"
  rm -rf "${LOGS}"

  mkdir -p "${DDIR}"
  mkdir -p "${LOGS}"
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

  echo "init Dusk node, done."
}

# start dusk mocks
start_dusk_mock_func() {
  echo "starting Dusk mocks $i ..."

#  EXEC_PID=$!
#  echo "started Dusk node, pid=$EXEC_PID"
}

# start dusk
start_dusk_func() {
  echo "starting Dusk node $i ..."

#  EXEC_PID=$!
#  echo "started Dusk node, pid=$EXEC_PID"
}

if [ "${INIT}" == "true" ]; then

  for i in $(seq 0 "$QTD"); do
    init_dusk_func "$i"
  done

fi

sleep 2

if [ "${MOCK}" == "true" ]; then
  for i in $(seq 0 "$QTD"); do
    start_dusk_mock_func "$i"
  done
fi

for i in $(seq 0 "$QTD"); do
  start_dusk_func "$i"
done

echo "done starting Dusk stack"
exit 0