# Dusk Blockchain Repository Overview

This intends to be a first-point-of-contact for developers looking into the
structure and looking for opportunities to contribute to the project, and to
help with the long term arc of migrating away from Go to Rust.

## Top level and repo related documents that don't fit elsewhere

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

[CONTRIBUTING.md](./CONTRIBUTING.md) Contribution guidelines for Dusk Golang
repositories.

[pkg/README.md](./pkg/README.md) Placeholder for something. Might be a nice site
for top level guide to the packages inside the repository.

[SUMMARY.md](./SUMMARY.md) Probably an earlier effort similar to this document,
mostly dead links.

## [cmd](./cmd)

Nothing directly exists in here, but the following items are the various tools
and services that make things happen.

## [cmd/deployer](./cmd/deployer)

Deployer is the name here as the totality of the full node on the Dusk network
is more than one component. Deployer is the launcher that coordinates the
startup and shutdown of Dusk and Rusk, the node and the VM. It monitors to see
they are still live, restarts them if needed, and shuts them down as required.

[cmd/deployer/README.md](./cmd/deployer/README.md) Notes about Deployer,
configuration and launch.

## [cmd/dusk](./cmd/dusk)

Dusk is the core blockchain and replication protocol.

It sets up the EventBus, RPC bus, blockchain core, gossip and kadcast peer to
peer node and connection to the Rusk VM. It handles all the basics, including
minting tokens, transfers between addresses, and then for everything else, the
data is processed and sent by the Rusk VM to be compiled into blocks. This of
course includes the Proof of Blind Bid leadeship selection for block production,
and of course, providing query services to a wallet for making transactions.

## [cmd/dusk/genesis](./cmd/dusk/genesis)

This just contains one function that prints the genesis block in JSON format.

## [cmd/netcollector](./cmd/netcollector)

A simple client that monitors transaction processing volumes via RPC to a Dusk
node.

## [cmd/utils](./cmd/utils)

In the top level here is a single interface to use the other tools within
folders, metrics, monitoring, and RPC access.

## [cmd/utils/grpcclient](./cmd/utils/grpcclient)

Provides staking automation and access to the live configuration system.

## [cmd/utils/metrics](./cmd/utils/metrics)

Provides information about various performance and data metrics of the node and
blockchain activity, blocks, pending transactions, and so on.

## [cmd/utils/mock](./cmd/utils/mock)

Provides mock versions of Rusk and Wallet and Transactor to use in tests for
generating dummy transactions and so forth.

## [cmd/utils/tps](./cmd/utils/tps)

A transaction generator specifically for stress testing transaction processing.

## [cmd/utils/transactions](./cmd/utils/transactions)

In here is a tool to test execution of transactions for staking and transfers.

## [cmd/voucher](./cmd/voucher)

A simple p2p voucher seeding node (what does that mean? It appears to mainly be
for finding and distributing the list of active peer addresses on the network)

## [cmd/voucher/challenger](./cmd/voucher/challenger)

This package provides the protocols for assessing a valid and correctly
functioning peer on the Dusk network

## [cmd/voucher/node](./cmd/voucher/node)

This is a peer information store used to keep and process information regarding
network peers and their status and addresses.

## [cmd/wallet](./cmd/wallet)

The entrypoint for running the Dusk CLI wallet (RPC client and keychain for
performing transactions)

## [cmd/wallet/conf](./cmd/wallet/conf)

This contains wallet configuration initialization and configuration, as well as
a network client dialing library that links node services to the external wallet
client. This dialer might be in a bad place.

## [cmd/wallet/prompt](./cmd/wallet/prompt)

Implements a menu driven interface for the CLI wallet, including staking and
transfers of tokens between addresses.

## [devnet-wallets](./devnet-wallets)

Contains a (rather large) set of wallets for working with devnet.

## [docs](./docs)

In here is one readme about using ELK.. elastic search thing?

[docs/elk.md](./docs/elk.md) Instructions for Elastic Stack on Docker, related
to using Kibana for log analysis

## [docs/internal](./docs/internal)

Internal documentation on details about Zenhub workflow for Dusk project
managers.

[docs/internal/zenhub.md](./docs/internal/zenhub.md) Guidelines for working with
Zenhub so our project managers lives are simpler.

## [docs/LICENSE](./docs/LICENSE)

Licences for used/modified code from btcd, golang (the compiler itself?) and
moneroutils

## [harness](./harness)

[harness.md](./harness.md) Some notes for working with the test harness

## [harness/engine](./harness/engine)

Contains the full test harnes implementation, dusk nodes with rusk mocks,
graphql and grpc endpoint, and managed simulated network between each spawned
node.

## [harness/tests](./harness/tests)

Tests to validate the test harness mocked node, transactions per second
benchmark, and simulated network.

## [harness/tests/node](./harness/tests/node)

Tests basic functionality, runs a node, makes simple queries, checks results are
consistent.

## [mocks](./mocks)

Mocks for testing things that work with Event, EventHandler and EventProcessor (
p2p/eventbus related?)

## [pkg](./pkg)

Just a couple of markdown documents here, standard idiomatic golang repository
libraries folder.

## [pkg/api](./pkg/api)

Top level http API interface for accessing EventBus, RPCBus and StormDB
services, internal and external messaging, and data storage.

## [pkg/config](./pkg/config)

Definitions, accessors and storage implementation for configuration of all
systems in dusk-blockchain.

[pkg/config.md](./pkg/config.md) Some documentation on the application
configuration system, and some example invocations

## [pkg/config/genesis](./pkg/config/genesis)

Specification for the genesis block data structure, a generator, and several
presets for different test networks.

## [pkg/config/samples](./pkg/config/samples)

Example configuration files and the default Dusk configuration file

## [pkg/core](./pkg/core)

Here can be found one document about the mempool high level overview.

[pkg/core/README.md](./pkg/core/README.md) Nothing here yet, but probably will
become a guide to the elements of core, which follow here in order.

[pkg/core/consensus/README.md](./pkg/core/consensus/README.md) Some notes about
the consensus and related specs and components.

[pkg/core/consensus/gotchas.md](./pkg/core/consensus/gotchas.md) Some notes
about listener ID values

[pkg/core/consensus/consensus.md](./pkg/core/consensus/consensus.md) A short
introduction to the SBA* consensus for Proof of Blind Bid selection of block
producers.

## [pkg/core/candidate](./pkg/core/candidate)

Block candidate validation and protocol implementation

## [pkg/core/chain](./pkg/core/chain)

Contains the Verifier, Loader and Ledger interfaces, mocks, the database store,
block synchronizer implementation.

[pkg/core/chain/README.md](./pkg/core/chain/README.md) A top level summary of
the architecture of the blockchain core

[pkg/core/chain/synchronizer.md](./pkg/core/chain/synchronizer.md)
Information about the block synchronisation, validation, storage and progress
reporting.

## [pkg/core/consensus](./pkg/core/consensus)

The interfaces and high level api's around the implementation of the SBA*
consensus, with lots of mocks and tests.

## [pkg/core/consensus/agreement](./pkg/core/consensus/agreement)

Accumulator, the score keeper of the agreement process, and the top level
protocol processing for performing a round of the consensus and agreement upon a
new block and its contents.

[pkg/core/consensus/agreement/agreement.md](./pkg/core/consensus/agreement/agreement.md)
High level description of the agreement phase, where consensus about a candidate
block is achieved in the network after candidates are reduced.

[pkg/core/consensus/agreement/README.md](./pkg/core/consensus/agreement/README.md)
Some architecture and notes about the agreement component in the consensus.

## [pkg/core/consensus/blockgenerator](./pkg/core/consensus/blockgenerator)

Top level interface and mock for the block generation process.

[pkg/core/consensus/blockgenerator/README.md](./pkg/core/consensus/blockgenerator/README.md)
Documentation of the block generator component and summary of the Blind Bid
protocol.

## [pkg/core/consensus/blockgenerator/candidate](./pkg/core/consensus/blockgenerator/candidate)

The interface, implementation and mocks for constructing a candidate block from
available transactions and provisioners.

[pkg/core/consensus/blockgenerator/candidate/readme.md](./pkg/core/consensus/blockgenerator/candidate/readme.md)
High level description of the candidate selection algorithm.

## [pkg/core/consensus/capi](./pkg/core/consensus/capi)

API for deploying a dusk consensus network, managing peers, rounds, members, and
stakes.

## [pkg/core/consensus/committee](./pkg/core/consensus/committee)

Handler for managing participation in the consensus comittee and inspecting the
membership and vote status.

## [pkg/core/consensus/header](./pkg/core/consensus/header)

Structure and implementation of the common consensus message header.

## [pkg/core/consensus/key](./pkg/core/consensus/key)

Implements a BLS12-381 key pair.

## [pkg/core/consensus/msg](./pkg/core/consensus/msg)

Verification of BLS signatures.

## [pkg/core/consensus/reduction](./pkg/core/consensus/reduction)

Implementation of the reduction phase that eliminates candidates in the
consensus agreement.

[pkg/core/consensus/reduction/README.md](./pkg/core/consensus/reduction/README.md)
High level description of the consensus candidate reduction algorithm.

[pkg/core/consensus/reduction/reduction.md](./pkg/core/consensus/reduction/reduction.md)
High level description of the binary reduction phase of the SBA* consensus.

## [pkg/core/consensus/reduction/firststep](./pkg/core/consensus/reduction/firststep)

Implementation of the first step of the reduction phase.

## [pkg/core/consensus/reduction/secondstep](./pkg/core/consensus/reduction/secondstep)

Implementation of the second step of the reduction phase.

[pkg/core/consensus/reduction/secondstep/gotchas.md](./pkg/core/consensus/reduction/secondstep/gotchas.md)
Notes regarding second step of consensus reduction

## [pkg/core/consensus/selection](./pkg/core/consensus/selection)

Implementation of selection phase that determines the winning bidder of the
Proof of Blind Bid winner and consensus block producer for the current block
height.

[pkg/core/consensus/selection/README.md](./pkg/core/consensus/selection/README.md)
Documentation on the selection algorithm at the centre of Proof of Blind Bid

[pkg/core/consensus/selection/selection.md](./pkg/core/consensus/selection/selection.md)
More notes about selection of block generators.

## [pkg/core/consensus/stakeautomaton](./pkg/core/consensus/stakeautomaton)

Watchdog process that automates the renewal of staking contracts the user
intends to continuously maintain.

## [pkg/core/consensus/testing](./pkg/core/consensus/testing)

Miscellaneous functions used in testing the consensus.

[pkg/core/consensus/testing/README.md](./pkg/core/consensus/testing/README.md)
The consensus integration testbed for testing the operation of the consensus in
a local cluster of nodes.

## [pkg/core/consensus/user](./pkg/core/consensus/user)

Managing the list of participating users, their roles and status in the
consensus.

[pkg/core/consensus/user/README.md](./pkg/core/consensus/user/README.md)
Some notes about the deterministic selection process.

[pkg/core/consensus/user/gotchas.md](./pkg/core/consensus/user/gotchas.md)
A note about iterating provisioner sets

## [pkg/core/data](./pkg/core/data)

Nothing here, underneath is codecs for various wire and storage formats of
blockchain, wallet and consensus data.

## [pkg/core/data/base58](./pkg/core/data/base58)

Encoding and decoding Base58 (no check) encoding.

## [pkg/core/data/block](./pkg/core/data/block)

Encoding, decoding and format of blocks, block headers and consensus
certificates.

## [pkg/core/data/database](./pkg/core/data/database)

Wrapper to K/V access for database back end and specific handling for
transactions.

[pkg/core/database/README.md](./pkg/core/database/README.md) Documentation for
the database interface used for storing data from flat files to key/value stores
for addresses, transactions, and the like

[pkg/core/database/heavy.md](./pkg/core/database/heavy.md) Notes about the
interface for bulky block data

[pkg/core/database/lite.md](./pkg/core/database/lite.md) Notes about the
memory-only database driver

## [pkg/core/data/ipc](./pkg/core/data/ipc)

This folder gathers IPC related items.

[pkg/core/data/ipc/transactions/gotchas.md](./pkg/core/data/ipc/transactions/gotchas.md)
Notes about the lack of formal canonical binary format, a year old.

## [pkg/core/data/ipc/common](./pkg/core/data/ipc/common)

Echo implementation to ping Rusk server.

## [pkg/core/data/ipc/keys](./pkg/core/data/ipc/keys)

Phoenix asymmetric cryptographic primitives, codec, signatures, validation,
address generation.

## [pkg/core/data/ipc/transactions](./pkg/core/data/ipc/transactions)

Codecs and accessors for all types of transaction messages used between Node and
Rusk VM.

## [pkg/core/data/txrecords](./pkg/core/data/txrecords)

A custom query filter aimed at simplifying a transaction information display for
user interfaces.

## [pkg/core/data/wallet](./pkg/core/data/wallet)

Wallet service with HD keychain and transaction index, minimal back end
interface.

## [pkg/core/database](./pkg/core/database)

An abstract driver interface for enabling multiple types of storage strategy
suited to different types of data, *and a blockchain data specific interface* -
with the purpose being data access workload optimised storage.

## [pkg/core/database/heavy](./pkg/core/database/heavy)

Database driver providing persistent storage via LevelDB.

## [pkg/core/database/lite](./pkg/core/database/lite)

Driver providing an ephemeral data store.

## [pkg/core/database/testing](./pkg/core/database/testing)

Mocks and tests for heavy and lite database drivers.

## [pkg/core/database/utils](./pkg/core/database/utils)

Miscellaneous helper functions used by other parts of pkg/core/database

## [pkg/core/loop](./pkg/core/loop)

State machine that drives, monitors and directs the processing during a round of
the consensus.

[pkg/core/loop/README.md](./pkg/core/loop/README.md) Some notes about the
consensus loop.

## [pkg/core/mempool](./pkg/core/mempool)

Implements the buffer for valid transactions not yet in a block. Includes an
optimised memory key/value data store and rebroadcaster.

[pkg/core/mempool.md](./pkg/core/mempool.md) A description of the salient
features and implementation details of the mempool, where pending transactions
are stored for block production and p2p distribution.

## [pkg/core/tests](./pkg/core/tests)

Nothing here.

## [pkg/core/tests/helper](./pkg/core/tests/helper)

A collection of helper functions for tests in pkg/core

## [pkg/core/transactor](./pkg/core/transactor)

An API for executing the various types of transactions that the node
understands. Staking, transfers, balances, and triggering smart contracts.

## [pkg/core/verifiers](./pkg/core/verifiers)

Verification functions for various sanity checks on blocks, coinbases and
consensus messages.

[pkg/core/verifiers/README.md](./pkg/core/verifiers/README.md) Some
documentation about context-free preliminary validation of data submitted to the
network (mainly checks on blocks)

## [pkg/gql](./pkg/gql)

A websocket and GraphQL enabled RPC for searching the blockchain database,
including subscriptions to notifications.

[pkg/gql/notifications.md](./pkg/gql/notifications.md) Some initial
documentation about notifications for RPC clients

[pkg/gql/README.md](./pkg/gql/README.md) Usage information about the GraphQL RPC
endpoints.

## [pkg/gql/notifications](./pkg/gql/notifications)

Notifications implementation, primarily distributes new accepted blocks to
subscribers, as well as broker-internal coordination messages.

## [pkg/gql/query](./pkg/gql/query)

Implements a simple human readable query syntax for requesting from blocks and
transactions specific fields and cross-cutting matches like address searches.

## [pkg/p2p](./pkg/p2p)

Some documentation on the p2p wire format, gossip and point to point messaging
architecture.

[pkg/p2p/README.md](./pkg/p2p/README.md) Notes about the design and
implementation of the peer to peer IO model.

## [pkg/p2p/kadcast](./pkg/p2p/kadcast)

The original implementation of the Kademlia Distributed Hash Table based
reliable broadcast routing protocol.

[pkg/p2p/kadcast/README.md](./pkg/p2p/kadcast/README.md) Documenting the Kadcast
protocol, specific to the Go implementation.

## [pkg/p2p/kadcast/encoding](./pkg/p2p/kadcast/encoding)

Message definitions and codecs for kadcast message types.

## [pkg/p2p/peer](./pkg/p2p/peer)

Peer to peer message sending and receiving, including one-to-many gossip message
distribution.

[pkg/p2p/peer/gotchas.md](./pkg/p2p/peer/gotchas.md) Notes regarding the
payload.Safe interface for message unmarshalling (decoding to runtime variables)

## [pkg/p2p/peer/dupemap](./pkg/p2p/peer/dupemap)

A cuckoo filter based deduplication engine.

## [pkg/p2p/peer/responding](./pkg/p2p/peer/responding)

Implementations of the various response types for queries between network peers.

## [pkg/p2p/wire](./pkg/p2p/wire)

Being otherwise empty, this *folder contains encoder specification document and
the Event and Store interfaces.*

[pkg/p2p/wire/README.md](./pkg/p2p/wire/README.md) Placeholder, currently.

[pkg/p2p/wire/encoding.md](./pkg/p2p/wire/encoding.md) Notes about the binary
codec. This might be deprecated by the use of rkyv?

[pkg/p2p/wire.md](./pkg/p2p/wire.md) Message data layouts for Dusk network peer
to peer messages.

## [pkg/p2p/wire/checksum](./pkg/p2p/wire/checksum)

Checksum algorithm used in wire messages to ensure error free reception. Based
on a truncated blake2b hash.

## [pkg/p2p/wire/encoding](./pkg/p2p/wire/encoding)

Encoding formats for integers and variable sized raw bytes.

## [pkg/p2p/wire/message](./pkg/p2p/wire/message)

Definitions of all message data structure formats and their wire/storage codecs,
mocks and tests.

## [pkg/p2p/wire/message/payload](./pkg/p2p/wire/message/payload)

This is actually just an interface for a concurrent safe deep copy function.

## [pkg/p2p/wire/protocol](./pkg/p2p/wire/protocol)

Protocol data transport formatting and network identifiers.

## [pkg/p2p/wire/topics](./pkg/p2p/wire/topics)

Definitions of the 1 byte message topic identifiers used in the wire encoding
and by EventBus

## [pkg/p2p/wire/util](./pkg/p2p/wire/util)

Helpers used in the pkg/p2p/wire package. In fact just has a function to get a
machine's Point of Presence on the internet, ie the gateway router of the LAN,
if its interface is not internet routable.

## [pkg/rpc](./pkg/rpc)

Contains a token based session authentication system.

## [pkg/rpc/client](./pkg/rpc/client)

Client interfaces for all of the gRPC based RPC and IPC interfaces used to
access and between Dusk blockchain and Rusk VM state machine.

## [pkg/rpc/server](./pkg/rpc/server)

Core gRPC server functionality with authentication and session management for
the clients in the client folder above.

## [pkg/util](./pkg/util)

There is notes here about the configuration system, based on viper.

## [pkg/util/container](./pkg/util/container)

## [pkg/util/container/ring](./pkg/util/container/ring)

A ring buffer that works with raw wire/storage encoded raw bytes data. Used in
the EventBus.

## [pkg/util/diagnostics](./pkg/util/diagnostics)

Instrumentation for using profiling with Dusk.

## [pkg/util/legacy](./pkg/util/legacy)

Converter that upgrades the old provisioners data structure to the new Rusk
Committee.

## [pkg/util/nativeutils](./pkg/util/nativeutils)

This is a large number of libraries from the pub/sub broker eventbus, logging,
error handling, a transmission protocol, two different set implementations.

## [pkg/util/nativeutils/eventbus](./pkg/util/nativeutils/eventbus)

This is a generic interface for interconnecting different components of Dusk
including separated runtimes like the Rusk VM

## [pkg/util/nativeutils/hashset](./pkg/util/nativeutils/hashset)

A concurrent safe hash set implementation.

## [pkg/util/nativeutils/logging](./pkg/util/nativeutils/logging)

Simplified setup and configuration for logrus.

## [pkg/util/nativeutils/prerror](./pkg/util/nativeutils/prerror)

Error type with priorities.

## [pkg/util/nativeutils/rcudp](./pkg/util/nativeutils/rcudp)

Simple short-message (max one UDP packet) reliable transport protocol using
Raptor FEC codes for retransmit-avoidance.

[pkg/util/nativeutils/rcudp/README.md](./pkg/util/nativeutils/rcudp/README.md)
Documentation regarding the Raptor codes based FEC used by the Gossip and
Kadcast components

## [pkg/util/nativeutils/rpcbus](./pkg/util/nativeutils/rpcbus)

Process-internal message passing system based on sending return channels in
requests.

## [pkg/util/nativeutils/sortedset](./pkg/util/nativeutils/sortedset)

Binary tree (which?) based sorted set for large numbers (such as hashes).

## [pkg/util/ruskmock](./pkg/util/ruskmock)

Rusk mock that can be programmed to give predefined or dummy responses to use in
tests of Dusk-Rusk communication.

[pkg/util/ruskmock/README.md](./pkg/util/ruskmock/README.md) Documentation on
the mock Rusk VM used for testing the core node.

## [scripts](./scripts)

Miscellaneous scripts such as a Go-based build script for Dusk and some launch
control scripts.
