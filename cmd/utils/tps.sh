#!/bin/bash

# example for usage:
# watch "./cmd/utils/tps.sh -q 4 -i 146512053 -p 8999"

SCRIPT=$(basename ${BASH_SOURCE[0]})
QTD=3
PORT=9000
ID=146512053

usage() {
  echo "Usage: $SCRIPT"
  echo "Optional command line arguments"
  echo "-q <number>  -- Quantity to run. eg: 3"
  echo "-i <number>  -- ID of node. eg: 146512053"
  echo "-p <number>  -- PORT of node. eg: 9000"

  exit 1
}

while getopts "h?q:i:p:" args; do
case $args in
    h|\?)
      usage;
      exit;;
    q ) QTD=${OPTARG};;
    i ) ID=${OPTARG};;
    p ) PORT=${OPTARG};;
  esac
done

set -euxo pipefail

sendtx_func() {
  echo "sending transactions to node $i ..."
  "$PWD"/bin/utils transactions --txtype=transfer --amount=1 --locktime=1 --grpcaddr=unix:///tmp/localnet-"${ID}"/node-$((PORT + i))/dusk-grpc.sock > /tmp/localnet-"${ID}"/tps.log 2>&1 & disown

  TX_PID=$!
  echo "sent transaction to node, pid=$TX_PID"
}

for i in $(seq 1 "$QTD"); do
  sendtx_func "$i"
done

echo "done sending transactions to Dusk nodes"
exit 0