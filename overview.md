# Dusk Blockchain Repository Overview

This intends to be a first-point-of-contact for developers looking into the
structure and looking for opportunities to contribute to the project, and to
help with the long term arc of migrating away from Go to Rust.

## All markdown documents found in this repository:

As a part of the documentation effort, all the existing documents found in the
repository are listed here in their lexicographic path order, with a brief
description of what is found inside.

[README.md](./README.md) The readme. Contains very basic getting started and
pointers to other info.

[logging.md](./logging.md) A set of notes containing example grep invocations to
filter the logs.

[.github/CODE_OF_CONDUCT.md](./.github/CODE_OF_CONDUCT.md) Our standards of
civility and manners, and mediation guidelines.

[.github/ISSUE_TEMPLATE/specs.md](./.github/ISSUE_TEMPLATE/specs.md)  Issue
template for specifications

[.github/ISSUE_TEMPLATE/feature_request.md](./.github/ISSUE_TEMPLATE/feature_request.md)
Issue template for feature requests

[.github/ISSUE_TEMPLATE/bug_report.md](./.github/ISSUE_TEMPLATE/bug_report.md)
Issue template for bug reports

[docs/README.md](./docs/README.md) Currently empty. Should be a for orienting
within docs folder.

[docs/internal/zenhub.md](./docs/internal/zenhub.md) Guidelines for working with
Zenhub so our project managers lives are simpler.

[docs/elk.md](./docs/elk.md) Instructions for Elastic Stack on Docker, related
to using Kibana for log analysis

[harness.md](./harness.md) Some notes for working with the test harness

[cmd/deployer/README.md](./cmd/deployer/README.md) Notes about Deployer,
configuration and launch.

[CONTRIBUTING.md](./CONTRIBUTING.md) Contribution guidelines for Dusk Golang
repositories.

[pkg/README.md](./pkg/README.md) Placeholder for something. Might be a nice site
for top level guide to the packages inside the repository.

[pkg/config.md](./pkg/config.md) Some documentation on the application
configuration system, and some example invocations

[pkg/gql/notifications.md](./pkg/gql/notifications.md) Some initial
documentation about notifications for RPC clients

[pkg/gql/README.md](./pkg/gql/README.md) Usage information about the GraphQL RPC
endpoints.

[pkg/core/database/README.md](./pkg/core/database/README.md) Documentation for
the database interface used for storing data from flat files to key/value stores
for addresses, transactions, and the like

[pkg/core/database/heavy.md](./pkg/core/database/heavy.md) Notes about the
interface for bulky block data

[pkg/core/database/lite.md](./pkg/core/database/lite.md) Notes about the
memory-only database driver

[pkg/core/README.md](./pkg/core/README.md) Nothing here yet, but probably will
become a guide to the elements of core, which follow here in order.

[pkg/core/data/ipc/transactions/gotchas.md](./pkg/core/data/ipc/transactions/gotchas.md)
Notes about the lack of formal canonical binary format, a year old.

[pkg/core/loop/README.md](./pkg/core/loop/README.md) Some notes about the 
consensus loop.

[pkg/core/verifiers/README.md](./pkg/core/verifiers/README.md) Some 
documentation about context-free preliminary validation of data submitted to 
the network (mainly checks on blocks)

[pkg/core/chain/README.md](./pkg/core/chain/README.md) A top level summary 
of the architecture of the blockchain core

[pkg/core/chain/synchronizer.md](./pkg/core/chain/synchronizer.md) 
Information about the block synchronisation, validation, storage and 
progress reporting.

[pkg/core/consensus/README.md](./pkg/core/consensus/README.md) Some notes 
about the consensus and related specs and components.

[pkg/core/consensus/gotchas.md](./pkg/core/consensus/gotchas.md) Some notes 
about listener ID values

[pkg/core/consensus/consensus.md](./pkg/core/consensus/consensus.md) A short 
introduction to the SBA* consensus for Proof of Blind Bid selection of block 
producers.

[pkg/core/consensus/selection/README.md](./pkg/core/consensus/selection/README.md)
Documentation on the selection algorithm at the centre of Proof of Blind Bid

[pkg/core/consensus/selection/selection.md](./pkg/core/consensus/selection/selection.md)
More notes about selection of block generators.

[pkg/core/consensus/testing/README.md](./pkg/core/consensus/testing/README.md)
The consensus integration testbed for testing the operation of the consensus 
in a local cluster of nodes.

[pkg/core/consensus/blockgenerator/README.md](./pkg/core/consensus/blockgenerator/README.md)
Documentation of the block generator component and summary of the Blind Bid 
protocol.

[pkg/core/consensus/blockgenerator/candidate/readme.md](./pkg/core/consensus/blockgenerator/candidate/readme.md)
High level description of the candidate selection algorithm.

[pkg/core/consensus/reduction/README.md](./pkg/core/consensus/reduction/README.md)
High level description of the consensus candidate reduction algorithm.

[pkg/core/consensus/reduction/reduction.md](./pkg/core/consensus/reduction/reduction.md)
High level description of the binary reduction phase of the SBA* consensus.

[pkg/core/consensus/reduction/secondstep/gotchas.md](./pkg/core/consensus/reduction/secondstep/gotchas.md)
Notes regarding second step of consensus reduction

[pkg/core/consensus/agreement/agreement.md](./pkg/core/consensus/agreement/agreement.md)
High level description of the agreement phase, where consensus about a 
candidate block is achieved in the network after candidates are reduced.

[pkg/core/consensus/agreement/README.md](./pkg/core/consensus/agreement/README.md)
Some architecture and notes about the agreement component in the consensus.

[pkg/core/consensus/user/README.md](./pkg/core/consensus/user/README.md)
Some notes about the deterministic selection process.

[pkg/core/consensus/user/gotchas.md](./pkg/core/consensus/user/gotchas.md)
A note about iterating provisioner sets

[pkg/core/mempool.md](./pkg/core/mempool.md) A description of the salient 
features and implementation details of the mempool, where pending 
transactions are stored for block production and p2p distribution.

[pkg/util/ruskmock/README.md](./pkg/util/ruskmock/README.md) Documentation 
on the mock Rusk VM used for testing the core node.

[pkg/util/nativeutils/rcudp/README.md](./pkg/util/nativeutils/rcudp/README.md)
Documentation regarding the Raptor codes based FEC used by the Gossip and 
Kadcast components

[pkg/p2p/wire/README.md](./pkg/p2p/wire/README.md) Placeholder, currently.

[pkg/p2p/wire/encoding.md](./pkg/p2p/wire/encoding.md) Notes about the 
binary codec. This might be deprecated by the use of rkyv?

[pkg/p2p/README.md](./pkg/p2p/README.md) Notes about the design and 
implementation of the peer to peer IO model.

[pkg/p2p/kadcast/README.md](./pkg/p2p/kadcast/README.md) Documenting the 
Kadcast protocol, specific to the Go implementation.

[pkg/p2p/wire.md](./pkg/p2p/wire.md) Message data layouts for Dusk network 
peer to peer messages.

[pkg/p2p/peer/gotchas.md](./pkg/p2p/peer/gotchas.md) Notes regarding the 
payload.Safe interface for message unmarshalling (decoding to runtime variables)

[SUMMARY.md](./SUMMARY.md) Probably an earlier effort similar to this 
document, mostly dead links.

## cmd

Nothing directly exists in here, but the following items are the various tools
and services that make things happen.

## cmd/deployer

Deployer is the name here as the totality of the full node on the Dusk network
is more than one component. Deployer is the launcher that coordinates the
startup and shutdown of Dusk and Rusk, the node and the VM. It monitors to see
they are still live, restarts them if needed, and shuts them down as required.

## cmd/dusk

Dusk is the core blockchain and replication protocol.

It sets up the EventBus, RPC bus, blockchain core, gossip and kadcast peer to
peer node and connection to the Rusk VM. It handles all the basics, including
minting tokens, transfers between addresses, and then for everything else, the
data is processed and sent by the Rusk VM to be compiled into blocks. This of
course includes the Proof of Blind Bid leadeship selection for block production,
and of course, providing query services to a wallet for making transactions.

## cmd/dusk/genesis

This just contains one function that prints the genesis block in JSON format.

## cmd/netcollector

A simple client that monitors transaction processing volumes via RPC to a Dusk
node.

## cmd/utils

In the top level here is a single interface to use the other tools within
folders, metrics, monitoring, and RPC access.

## cmd/utils/grpcclient

Provides staking automation and access to the live configuration system.

## cmd/utils/metrics

Provides information about various performance and data metrics of the node and
blockchain activity, blocks, pending transactions, and so on.

## cmd/utils/mock

Provides mock versions of Rusk and Wallet and Transactor to use in tests for
generating dummy transactions and so forth.

## cmd/utils/tps

A transaction generator specifically for stress testing transaction processing.

## cmd/utils/transactions

In here is a tool to test execution of transactions for staking and transfers.

## cmd/voucher

A simple p2p voucher seeding node (what does that mean? It appears to mainly be for finding and distributing the list of active peer addresses on the network)

## cmd/voucher/challenger

This package provides the protocols for assessing a valid and correctly functioning peer on the Dusk network

## cmd/voucher/node

This is a peer information store used to keep and process information regarding network peers and their status and addresses.

## cmd/wallet

The entrypoint for running the Dusk CLI wallet (RPC client and keychain for performing transactions)

## cmd/wallet/conf

This contains wallet configuration initialization and configuration, as well as a network client dialing library that links node services to the external wallet client. This dialer might be in a bad place.

## cmd/wallet/prompt

Implements a menu driven interface for the CLI wallet, including staking and transfers of tokens between addresses.

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
