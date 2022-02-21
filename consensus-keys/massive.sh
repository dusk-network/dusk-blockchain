#!/bin/bash

for i in `seq 1 99`;
do
    RUSK_WALLET_PWD=password $RUSK_PROFILE_PATH/target/release/rusk-wallet --skip-recovery --wallet-file wallets/node_$i create
    RUSK_WALLET_PWD=password $RUSK_PROFILE_PATH/target/release/rusk-wallet --wallet-file wallets/node_$i --data-dir wallets export --key 0
    RUSK_WALLET_PWD=password $RUSK_PROFILE_PATH/target/release/rusk-wallet --wallet-file wallets/node_$i  address --key 0 | sed 's/> Public address for key 0 is: //g' >  wallets/node_$i.address
    mv wallets/node_$i-0.key $HOME/go/src/github.com/dusk-network/dusk-blockchain/consensus-keys/node_$i.keys
    mv wallets/node_$i-0.pub $HOME/go/src/github.com/dusk-network/dusk-blockchain/consensus-keys/node_$i.pub
    mv wallets/node_$i.address $HOME/go/src/github.com/dusk-network/dusk-blockchain/consensus-keys/
    mv wallets/node_$i.dat $HOME/go/src/github.com/dusk-network/dusk-blockchain/consensus-keys/
done
