#!/bin/bash

SCRIPT=$(basename ${BASH_SOURCE[0]})
QTD=2 #QTD=2 == 3 nodes since it counts from zero
TYPE=branch
DUSK_VERSION=v0.3.0
INIT=true
VOUCHER=true
SEND_BID=true

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

  exit 1
}

while getopts "h?c:q:s:i:" args; do
case $args in
    h|\?)
      usage;
      exit;;
    c ) TYPE=${OPTARG};;
    q ) QTD=${OPTARG};;
    s ) DUSK_VERSION=${OPTARG};;
    i ) INIT=${OPTARG};;
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

[gql]
  address = "127.0.0.1:$((9500+$i))"
  enabled = "true"
  network = "tcp"

[logger]
  output = "${DDIR}/dusk"
  level = "debug"

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

[wallet]
  file = "${currentDir}/harness/data/wallet-$((9000+$i)).dat"
  store = "${DDIR}/walletDB/"
EOF

  echo "init Dusk node $i, done."
}

# start dusk
start_dusk_func() {
  echo "starting Dusk node $i ..."

  DDIR="${currentDir}/devnet/dusk_data/dusk${i}"

  CMD="${currentDir}/bin/dusk --config ${DDIR}/dusk.toml"
  TMPDIR=$DDIR ${CMD} >> "${currentDir}/devnet/dusk_data/logs/dusk$i.log" 2>&1 &

  EXEC_PID=$!
  echo "started Dusk node $i, pid=$EXEC_PID"
  sleep 1

  # load wallet cmd
  LOADWALLET_CMD="./bin/utils walletutils --grpchost unix://${DDIR}/dusk-grpc.sock --walletcmd loadwallet --walletpassword password"
  ${LOADWALLET_CMD} >> "${currentDir}/devnet/dusk_data/logs/load_wallet$i.log" 2>&1 &

}

# start blindbid
start_blindbid_func() {
  echo "starting Blindbid node $i ..."

  DDIR="${currentDir}/devnet/dusk_data/dusk${i}"

  CMD="${currentDir}/bin/blindbid-linux-amd64"
  TMPDIR=$DDIR ${CMD} >> "${currentDir}/devnet/dusk_data/logs/blindbid$i.log" 2>&1 &

  EXEC_PID=$!
  echo "started Blindbid node $i, pid=$EXEC_PID"

}

# send_bid_func
send_bid_func(){
  echo "Sending bid to node $i ..."

  # send bid cmd
  SENDBID_CMD="./bin/utils transactions --grpchost unix://${DDIR}/dusk-grpc.sock --txtype consensus --amount 10 --locktime 10"
  ${SENDBID_CMD} >> "${currentDir}/devnet/dusk_data/logs/sendbid_bid$i.log" 2>&1 &
}

# start_voucher_func
start_voucher_func() {
  echo "starting Dusk Voucher ..."

  CMD="${currentDir}/bin/voucher"
  ${CMD} >> "${currentDir}/devnet/dusk_data/logs/voucher.log" 2>&1 &

  EXEC_PID=$!
  echo "started Dusk Voucher, pid=$EXEC_PID"
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

for i in $(seq 0 "$QTD"); do
  start_blindbid_func "$i"
done

for i in $(seq 0 "$QTD"); do
  start_dusk_func "$i"
done

sleep 5

if [ "${SEND_BID}" == "true" ]; then

  for i in $(seq 0 "$QTD"); do
    send_bid_func "$i"
  done

fi


echo "done starting Dusk stack"
exit 0