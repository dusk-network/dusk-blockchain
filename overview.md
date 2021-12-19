# Dusk Blockchain Repository Overview

This intends to be a first-point-of-contact for developers looking into
the structure and looking for opportunities to contribute to the project, 
and to help with the long term arc of migrating away from Go to Rust.

## All markdown documents found in this repository:

As a part of the documentation effort, all the existing documents found in the
repository are listed here in their lexicographic path order, with a brief 
description of what is found inside.

[README.md](./README.md)

[logging.md](./logging.md)

[.github/CODE_OF_CONDUCT.md](./.github/CODE_OF_CONDUCT.md)

[.github/ISSUE_TEMPLATE/specs.md](./.github/ISSUE_TEMPLATE/specs.md)

[.github/ISSUE_TEMPLATE/feature_request.md](./.github/ISSUE_TEMPLATE/feature_request.md)

[.github/ISSUE_TEMPLATE/bug_report.md](./.github/ISSUE_TEMPLATE/bug_report.md)

[docs/README.md](./docs/README.md)

[docs/internal/zenhub.md](./docs/internal/zenhub.md)

[docs/elk.md](./docs/elk.md)

[harness.md](./harness.md)

[cmd](./cmd)

[cmd/deployer/README.md](./cmd/deployer/README.md)

[CONTRIBUTING.md](./CONTRIBUTING.md)

[pkg/README.md](./pkg/README.md)

[pkg/config.md](./pkg/config.md)

[pkg/gql/notifications.md](./pkg/gql/notifications.md)

[pkg/gql/README.md](./pkg/gql/README.md)

[pkg/core/database/README.md](./pkg/core/database/README.md)

[pkg/core/database/heavy.md](./pkg/core/database/heavy.md)

[pkg/core/database/lite.md](./pkg/core/database/lite.md)

[pkg/core/README.md](./pkg/core/README.md)

[pkg/core/data/ipc/transactions/gotchas.md](./pkg/core/data/ipc/transactions/gotchas.md)

[pkg/core/loop/README.md](./pkg/core/loop/README.md)

[pkg/core/verifiers/README.md](./pkg/core/verifiers/README.md)

[pkg/core/chain/README.md](./pkg/core/chain/README.md)

[pkg/core/chain/synchronizer.md](./pkg/core/chain/synchronizer.md)

[pkg/core/consensus/README.md](./pkg/core/consensus/README.md)

[pkg/core/consensus/gotchas.md](./pkg/core/consensus/gotchas.md)

[pkg/core/consensus/consensus.md](./pkg/core/consensus/consensus.md)

[pkg/core/consensus/selection/README.md](./pkg/core/consensus/selection/README.md)

[pkg/core/consensus/selection/selection.md](./pkg/core/consensus/selection/selection.md)

[pkg/core/consensus/testing/README.md](./pkg/core/consensus/testing/README.md)

[pkg/core/consensus/blockgenerator/README.md](./pkg/core/consensus/blockgenerator/README.md)

[pkg/core/consensus/blockgenerator/candidate/readme.md](./pkg/core/consensus/blockgenerator/candidate/readme.md)

[pkg/core/consensus/reduction/README.md](./pkg/core/consensus/reduction/README.md)

[pkg/core/consensus/reduction/reduction.md](./pkg/core/consensus/reduction/reduction.md)

[pkg/core/consensus/reduction/secondstep/gotchas.md](./pkg/core/consensus/reduction/secondstep/gotchas.md)

[pkg/core/consensus/agreement/agreement.md](./pkg/core/consensus/agreement/agreement.md)

[pkg/core/consensus/agreement/README.md](./pkg/core/consensus/agreement/README.md)

[pkg/core/consensus/user/README.md](./pkg/core/consensus/user/README.md)

[pkg/core/consensus/user/gotchas.md](./pkg/core/consensus/user/gotchas.md)

[pkg/core/mempool.md](./pkg/core/mempool.md)

[pkg/util/ruskmock/README.md](./pkg/util/ruskmock/README.md)

[pkg/util/nativeutils/rcudp/README.md](./pkg/util/nativeutils/rcudp/README.md)

[pkg/p2p/wire/README.md](./pkg/p2p/wire/README.md)

[pkg/p2p/wire/encoding.md](./pkg/p2p/wire/encoding.md)

[pkg/p2p/README.md](./pkg/p2p/README.md)

[pkg/p2p/kadcast/README.md](./pkg/p2p/kadcast/README.md)

[pkg/p2p/wire.md](./pkg/p2p/wire.md)

[pkg/p2p/peer/gotchas.md](./pkg/p2p/peer/gotchas.md)

[SUMMARY.md](./SUMMARY.md)

[overview.md](./overview.md)



## cmd

Nothing directly exists in here, but the following items are the various
tools and services that make things happen.

## cmd/deployer

Deployer is the name here as the totality of the full node on the Dusk network is more than one component.
Deployer is the launcher that coordinates the startup and shutdown of Dusk and Rusk, the node and the VM.
It monitors to see they are still live, restarts them if needed, and shuts them down as required.

## cmd/dusk

Dusk is the core blockchain and replication protocol.

It sets up the EventBus, RPC bus, blockchain core, gossip and kadcast peer to peer node and connection to 
the Rusk VM. It handles all the basics, including minting tokens, transfers between addresses, and then 
for everything else, the data is processed and sent by the Rusk VM to be compiled into blocks. This of 
course includes the Proof of Blind Bid leadeship selection for block production, and of course, providing
query services to a wallet for making transactions.

## cmd/dusk/genesis

This just contains one function that prints the genesis block in JSON format.

## cmd/netcollector

A simple client that monitors transaction processing volumes via RPC to a Dusk node.

## cmd/utils

In the top level here is a single interface to use the other tools within folders, metrics, monitoring, 
and RPC access.

## cmd/utils/grpcclient

Provides staking automation and access to the live configuration system.

## cmd/utils/metrics

Provides information about various performance and data metrics of the node and blockchain activity, blocks,
pending transactions, and so on.

## cmd/utils/mock

Provides mock versions of Rusk and Wallet and Transactor to use in tests for generating dummy transactions
and so forth.

## cmd/utils/tps

A transaction generator specifically for stress testing transaction processing.

## cmd/utils/transactions

In here is a tool to test execution of transactions for staking and transfers.

## cmd/voucher



## cmd/voucher/challenger

## cmd/voucher/node

## cmd/wallet

## cmd/wallet/conf

## cmd/wallet/prompt

## devnet-wallets

## docs

## docs/internal

## docs/LICENSE

## harness

## harness/engine

## harness/tests

## harness/tests/node

## mocks

## pkg

## pkg/api

## pkg/config

## pkg/config/genesis

## pkg/config/samples

## pkg/core

## pkg/core/candidate

## pkg/core/chain

## pkg/core/consensus

## pkg/core/consensus/agreement

## pkg/core/consensus/blockgenerator

## pkg/core/consensus/blockgenerator/candidate

## pkg/core/consensus/capi

## pkg/core/consensus/committee

## pkg/core/consensus/header

## pkg/core/consensus/key

## pkg/core/consensus/msg

## pkg/core/consensus/reduction

## pkg/core/consensus/reduction/firststep

## pkg/core/consensus/reduction/secondstep

## pkg/core/consensus/selection

## pkg/core/consensus/stakeautomaton

## pkg/core/consensus/testing

## pkg/core/consensus/user

## pkg/core/data

## pkg/core/data/base58

## pkg/core/data/block

## pkg/core/data/database

## pkg/core/data/ipc

## pkg/core/data/ipc/common

## pkg/core/data/ipc/keys

## pkg/core/data/ipc/transactions

## pkg/core/data/txrecords

## pkg/core/data/wallet

## pkg/core/database

## pkg/core/database/heavy

## pkg/core/database/lite

## pkg/core/database/testing

## pkg/core/database/utils

## pkg/core/loop

## pkg/core/mempool

## pkg/core/tests

## pkg/core/tests/helper

## pkg/core/transactor

## pkg/core/verifiers

## pkg/gql

## pkg/gql/notifications

## pkg/gql/query

## pkg/p2p

## pkg/p2p/kadcast

## pkg/p2p/kadcast/encoding

## pkg/p2p/peer

## pkg/p2p/peer/dupemap

## pkg/p2p/peer/responding

## pkg/p2p/wire

## pkg/p2p/wire/checksum

## pkg/p2p/wire/encoding

## pkg/p2p/wire/message

## pkg/p2p/wire/message/payload

## pkg/p2p/wire/protocol

## pkg/p2p/wire/topics

## pkg/p2p/wire/util

## pkg/rpc

## pkg/rpc/client

## pkg/rpc/server

## pkg/util

## pkg/util/container

## pkg/util/container/ring

## pkg/util/diagnostics

## pkg/util/legacy

## pkg/util/nativeutils

## pkg/util/nativeutils/eventbus

## pkg/util/nativeutils/hashset

## pkg/util/nativeutils/logging

## pkg/util/nativeutils/prerror

## pkg/util/nativeutils/rcudp

## pkg/util/nativeutils/rpcbus

## pkg/util/nativeutils/sortedset

## pkg/util/ruskmock

## scripts

106 directories
