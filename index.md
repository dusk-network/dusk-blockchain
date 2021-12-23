# [github.com/dusk-network/dusk-blockchain](https://github.com/dusk-network/dusk-blockchain)

## Index of Documentation

This document is constructed automatically with the help of the use of a 
template format for the primary `README.md` of a directory in a repository 
in order to serve as a means to locate something by the keywords appearing 
in the document headers for all directories in the repository.

---

### [.github/ISSUE_TEMPLATE/README.md](./.github/ISSUE_TEMPLATE/README.md)

Here are the templates that define the three main types of issues.
### [.github/workflows/README.md](./.github/workflows/README.md)

These are continuous integration/continuous deployment scripts for Github for
validating changes before merging to the main branch.
### [README.md](./README.md)

Official reference implementation of the DUSK Network protocol in Golang.

[![Actions Status](https://github.com/dusk-network/dusk-blockchain/workflows/Continuous%20Integration/badge.svg)](https://github.com/dusk-network/dusk-blockchain/actions)
[![codecov](https://codecov.io/gh/dusk-network/dusk-blockchain/branch/master/graph/badge.svg)](https://codecov.io/gh/dusk-network/dusk-blockchain)
[![Go Report Card](https://goreportcard.com/badge/github.com/dusk-network/dusk-blockchain?style=flat-square)](https://goreportcard.com/report/github.com/dusk-network/dusk-blockchain)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/dusk-network/dusk-blockchain)](https://pkg.go.dev/github.com/dusk-network/dusk-blockchain)
### [cmd/README.md](./cmd/README.md)

Nothing directly exists in here, but the following items are the various tools
and services that make things happen.
### [cmd/deployer/README.md](./cmd/deployer/README.md)

Deployer is an application that simplifies the procedure of (re)starting a
dusk-blockchain node (both dusk-blockchain and rusk services). It should also
facilitate automatic diagnostic of runtime issues.
### [cmd/dusk/README.md](./cmd/dusk/README.md)

Dusk is the main entrypoint for starting up a Dusk blockchain node, which
provides the dusk consensus for block producers, synchronisation, transaction
relaying and processing. Smart contracts and the global network state are
offloaded to the Rusk VM.
### [cmd/dusk/genesis/README.md](./cmd/dusk/genesis/README.md)

This just contains one function that prints the genesis block in JSON format.
### [cmd/netcollector/README.md](./cmd/netcollector/README.md)

A simple client that monitors transaction processing volumes via RPC to a Dusk
node.
### [cmd/utils/README.md](./cmd/utils/README.md)

In the top level here is a single interface to use the other tools within
folders, metrics, monitoring, and RPC access.
### [cmd/utils/grpcclient/README.md](./cmd/utils/grpcclient/README.md)

Provides staking automation and access to the live configuration system.
### [cmd/utils/metrics/README.md](./cmd/utils/metrics/README.md)

Provides information about various performance and data metrics of the node and
blockchain activity, blocks, pending transactions, and so on.
### [cmd/utils/mock/README.md](./cmd/utils/mock/README.md)

Provides mock versions of Rusk and Wallet and Transactor to use in tests for
generating dummy transactions and so forth.
### [cmd/utils/tps/README.md](./cmd/utils/tps/README.md)

A transaction generator specifically for stress testing transaction processing.
### [cmd/utils/transactions/README.md](./cmd/utils/transactions/README.md)

In here is a tool to test execution of transactions for staking and transfers.
### [cmd/voucher/README.md](./cmd/voucher/README.md)

A simple p2p voucher seeding node (what does that mean? It appears to mainly be
for finding and distributing the list of active peer addresses on the network)
### [cmd/voucher/challenger/README.md](./cmd/voucher/challenger/README.md)

This package provides the protocols for assessing a valid and correctly
functioning peer on the Dusk network
### [cmd/voucher/node/README.md](./cmd/voucher/node/README.md)

This is a peer information store used to keep and process information regarding
network peers and their status and addresses.
### [cmd/wallet/README.md](./cmd/wallet/README.md)

The entrypoint for running the Dusk CLI wallet (RPC client and keychain for
performing transactions)
### [cmd/wallet/conf/README.md](./cmd/wallet/conf/README.md)

This contains wallet configuration initialization and configuration, as well as
a network client dialing library that links node services to the external wallet
client. This dialer might be in a bad place.
### [cmd/wallet/prompt/README.md](./cmd/wallet/prompt/README.md)

Implements a menu driven interface for the CLI wallet, including staking and
transfers of tokens between addresses.
### [devnet-wallets/README.md](./devnet-wallets/README.md)

Contains a (rather large) set of wallets for working with devnet.
### [docs/LICENSE/README.md](./docs/LICENSE/README.md)

Licences for used/modified code from btcd, golang (the compiler itself?) and
moneroutils
### [docs/README.md](./docs/README.md)

This will be the central point from which users can orient in order to discover
the relevant documentation for their questions and purposes.
### [docs/internal/README.md](./docs/internal/README.md)

Internal documentation on details about Zenhub workflow for Dusk project
managers.
### [docs/template.README.md](./docs/template.README.md)

Blurb about what this package does in one sentence here. Optional badges and
multiple paragraphs but it may belong in an Introduction if it is long.
### [harness/README.md](./harness/README.md)

Test harness for simulation network testing.
### [harness/engine/README.md](./harness/engine/README.md)

Contains the full test harnes implementation, dusk nodes with rusk mocks,
graphql and grpc endpoint, and managed simulated network between each spawned
node.
### [harness/tests/README.md](./harness/tests/README.md)

Tests to validate the test harness mocked node, transactions per second
benchmark, and simulated network.
### [harness/tests/node/README.md](./harness/tests/node/README.md)

Tests basic functionality, runs a node, makes simple queries, checks results are
consistent.
### [mocks/README.md](./mocks/README.md)

Mocks for testing things that work with Event, EventHandler and EventProcessor (
p2p/eventbus related?)
### [pkg/README.md](./pkg/README.md)

In accordance with Go standard repository layout conventions, this folder
contains all of the libraries used primarily internally by applications in
the [cmd/]() folder.
### [pkg/api/README.md](./pkg/api/README.md)

Top level http API interface for accessing EventBus, RPCBus and StormDB
services, internal and external messaging, and data storage.
### [pkg/config/README.md](./pkg/config/README.md)

Definitions, accessors and storage implementation for configuration of all
systems in dusk-blockchain.
### [pkg/config/genesis/README.md](./pkg/config/genesis/README.md)

Specification for the genesis block data structure, a generator, and several
presets for different test networks.
### [pkg/config/samples/README.md](./pkg/config/samples/README.md)

Example configuration files and the default Dusk configuration file
### [pkg/core/README.md](./pkg/core/README.md)

This folder contains all of the libraries that implement the functionality of
everything that interacts with the internals of the Node - consensus, storage
and validation.
### [pkg/core/candidate/README.md](./pkg/core/candidate/README.md)

Block candidate protocol and validation implementation.
### [pkg/core/chain/README.md](./pkg/core/chain/README.md)

The `Chain` is a component which is fully responsible for building the
blockchain. It judges the validity of incoming blocks, appends them to the
database, and disseminates newly approved blocks both internally (through the
event bus) and externally (over the gossip network).

Additionally, it is aware of the node's state in the network at all times, and
controls the execution of the consensus by responding to perceived state
changes.
### [pkg/core/consensus/README.md](./pkg/core/consensus/README.md)

The interfaces and high level api's around the implementation of the SBA*
consensus, with lots of mocks and tests.
### [pkg/core/consensus/agreement/README.md](./pkg/core/consensus/agreement/README.md)

This package implements the component which performs
the [Agreement Phase](./agreement.md) of the consensus.
### [pkg/core/consensus/blockgenerator/README.md](./pkg/core/consensus/blockgenerator/README.md)

This package implements a full block generator component, to be used by
participants of the blind-bid protocol in the SBA\* consensus. It is internally
made up of two distinct components, a [score generator](./score/README.md), and
a [candidate generator](./candidate/README.md).
### [pkg/core/consensus/blockgenerator/candidate/README.md](./pkg/core/consensus/blockgenerator/candidate/README.md)

This package implements a candidate generator, for the blind-bid protocol of the
SBA\* consensus protocol.
### [pkg/core/consensus/capi/README.md](./pkg/core/consensus/capi/README.md)

API for deploying a dusk consensus network, managing peers, rounds, members, and
stakes.
### [pkg/core/consensus/committee/README.md](./pkg/core/consensus/committee/README.md)

Handler for managing participation in the consensus comittee and inspecting the
membership and vote status.
### [pkg/core/consensus/header/README.md](./pkg/core/consensus/header/README.md)

Structure and implementation of the common consensus message header.
### [pkg/core/consensus/key/README.md](./pkg/core/consensus/key/README.md)

Implements a BLS12-381 key pair.
### [pkg/core/consensus/msg/README.md](./pkg/core/consensus/msg/README.md)

Verification of BLS signatures.
### [pkg/core/consensus/reduction/README.md](./pkg/core/consensus/reduction/README.md)

This package defines the components for both the first and second step of
reduction individually, as specified in
the [Binary Reduction Phase](./reduction.md) of the SBA\* consensus protocol.

Due to small, but intricate differences between the two steps, the components
are defined individually, with a minimal amount of shared code. This was done in
an effort to increase readability, since it avoids massive amounts of
abstraction which, on earlier iterations of the package, caused some confusion
for readers. That said, code which is identical across the two components is
defined in the top-level of the package, and imported down.
### [pkg/core/consensus/reduction/firststep/README.md](./pkg/core/consensus/reduction/firststep/README.md)

Implementation of the first step of the reduction phase.
### [pkg/core/consensus/reduction/secondstep/README.md](./pkg/core/consensus/reduction/secondstep/README.md)

Implementation of the second step of the reduction phase.
### [pkg/core/consensus/selection/README.md](./pkg/core/consensus/selection/README.md)

The `Selector` is the component appointed to collect the scores, verify the _
zero-knowledge proof_ thereto associated and to propagate the _block hash_
associated with the highest score observed during a period of time denoted
as `timeLength`. The _block hash_ is then forwarded to the `EventBus` to be
picked up by the `Block Reducer`

An abstract description of the Selection phase in the SBA\* consensus protocol
can be found [here](./selection.md).
### [pkg/core/consensus/stakeautomaton/README.md](./pkg/core/consensus/stakeautomaton/README.md)

Watchdog process that automates the renewal of staking contracts the user
intends to continuously maintain.
### [pkg/core/consensus/testing/README.md](./pkg/core/consensus/testing/README.md)

Miscellaneous functions used in testing the consensus.
### [pkg/core/consensus/user/README.md](./pkg/core/consensus/user/README.md)

This package implements the data structure which holds the Provisioner
committee, and implements methods on top of this committee in order to be able
to extract **voting committees** which are eligible to decide on blocks during
the SBA\* consensus protocol.
### [pkg/core/data/README.md](./pkg/core/data/README.md)

Definitions of data types and interfaces used in the core components and their
codecs for wire and storage purposes.
### [pkg/core/data/base58/README.md](./pkg/core/data/base58/README.md)

Encoding and decoding Base58 (no check) encoding.
### [pkg/core/data/block/README.md](./pkg/core/data/block/README.md)

Encoding, decoding and format of blocks, block headers and consensus
certificates.
### [pkg/core/data/database/README.md](./pkg/core/data/database/README.md)

Wrapper to K/V access for database back end and specific handling for
transactions.
### [pkg/core/data/ipc/README.md](./pkg/core/data/ipc/README.md)

This folder gathers IPC related items.
### [pkg/core/data/ipc/common/README.md](./pkg/core/data/ipc/common/README.md)

Echo implementation to ping Rusk server.
### [pkg/core/data/ipc/keys/README.md](./pkg/core/data/ipc/keys/README.md)

Phoenix asymmetric cryptographic primitives, codec, signatures, validation,
address generation.
### [pkg/core/data/ipc/transactions/README.md](./pkg/core/data/ipc/transactions/README.md)

Codecs and accessors for all types of transaction messages used between Node and
Rusk VM.
### [pkg/core/data/txrecords/README.md](./pkg/core/data/txrecords/README.md)

A custom query filter aimed at simplifying a transaction information display for
user interfaces.
### [pkg/core/data/wallet/README.md](./pkg/core/data/wallet/README.md)

Wallet service with HD keychain and transaction index, minimal back end
interface.
### [pkg/core/database/README.md](./pkg/core/database/README.md)

An abstract driver interface for enabling multiple types of storage strategy
suited to different types of data, *and a blockchain data specific interface* -
with the purpose being data access workload optimised storage. TODO: this merges
two independent incomplete documents with the same intent, even a header is
duplicated.
### [pkg/core/database/heavy/README.md](./pkg/core/database/heavy/README.md)

Database driver providing persistent storage via LevelDB.
### [pkg/core/database/lite/README.md](./pkg/core/database/lite/README.md)

Driver providing an ephemeral data store.
### [pkg/core/database/testing/README.md](./pkg/core/database/testing/README.md)

Mocks and tests for heavy and lite database drivers.
### [pkg/core/database/utils/README.md](./pkg/core/database/utils/README.md)

Miscellaneous helper functions used by other parts of pkg/core/database.
### [pkg/core/loop/README.md](./pkg/core/loop/README.md)

The `Loop` is the most high-level component of the SBA\* consensus
implementation. It simulates a state-machine like architecture, with a
functional flavor, where consensus phases are sequenced one after the other,
returning functions as they complete, based on their outcomes.
### [pkg/core/mempool/README.md](./pkg/core/mempool/README.md)

Package mempool represents the chain transaction layer \(not to be confused with
DB transaction layer\). A blockchain transaction has one of a finite number of
states at any given time. Below are listed the logical states of a transaction.
### [pkg/core/tests/README.md](./pkg/core/tests/README.md)

Things for tests. [helper](./helper/) contains helpers for tests.
### [pkg/core/tests/helper/README.md](./pkg/core/tests/helper/README.md)

A collection of helper functions for tests in pkg/core
### [pkg/core/transactor/README.md](./pkg/core/transactor/README.md)

An API for executing the various types of transactions that the node
understands. Staking, transfers, balances, and triggering smart contracts.
### [pkg/core/verifiers/README.md](./pkg/core/verifiers/README.md)

Verification functions for various sanity checks on blocks, coinbases and
consensus messages.
### [pkg/gql/README.md](./pkg/gql/README.md)

A websocket and GraphQL enabled RPC for searching the blockchain database,
including subscriptions to notifications.
### [pkg/gql/notifications/README.md](./pkg/gql/notifications/README.md)

Notification system to broadcast updates to subscribed websocket clients. It can
be considered as a bridge between EventBus events and websocket connections. In
most cases, a subscriber would be the UI client expecting updates on new block
accepted. Additionally, this package could be extended to provide subscriptions
for more specific events \(e.g a client needs a notification on a specific
tx\_hash accepted\)
### [pkg/gql/query/README.md](./pkg/gql/query/README.md)

Implements a simple human readable query syntax for requesting from blocks and
transactions specific fields and cross-cutting matches like address searches.
### [pkg/p2p/README.md](./pkg/p2p/README.md)

Some documentation on the p2p wire format, gossip and point to point messaging
architecture.
### [pkg/p2p/kadcast/README.md](./pkg/p2p/kadcast/README.md)

The original implementation of the Kademlia Distributed Hash Table based
reliable broadcast routing protocol.
### [pkg/p2p/kadcast/encoding/README.md](./pkg/p2p/kadcast/encoding/README.md)

Message definitions and codecs for kadcast message types.
### [pkg/p2p/peer/README.md](./pkg/p2p/peer/README.md)

Peer to peer message sending and receiving, including one-to-many gossip message
distribution.
### [pkg/p2p/peer/dupemap/README.md](./pkg/p2p/peer/dupemap/README.md)

A cuckoo filter based deduplication engine.
### [pkg/p2p/peer/responding/README.md](./pkg/p2p/peer/responding/README.md)

Implementations of the various response types for queries between network peers.
### [pkg/p2p/wire/README.md](./pkg/p2p/wire/README.md)

Being otherwise empty, this *folder also contains encoder specification document
and the Event and Store interfaces.*
### [pkg/p2p/wire/checksum/README.md](./pkg/p2p/wire/checksum/README.md)

Checksum algorithm used in wire messages to ensure error free reception. Based
on a truncated blake2b hash.
### [pkg/p2p/wire/encoding/README.md](./pkg/p2p/wire/encoding/README.md)

The encoding library provides methods to serialize different data types for
storage and transfer over the wire protocol. The library wraps around the
standard `Read` and `Write` functions of an `io.Reader` or `io.Writer`, to
provide simple methods for any basic data type. The library is built to mimic
the way the `binary.Read` and `binary.Write` functions work, but aims to
simplify the code to gain processing speed, and avoid reflection-based
encoding/decoding. As a general rule, for encoding a variable, you should pass
it by value, and for decoding a value, you should pass it by reference. For an
example of this, please refer to the tests or any implementation of this library
in the codebase.
### [pkg/p2p/wire/message/README.md](./pkg/p2p/wire/message/README.md)

Definitions of all message data structure formats and their wire/storage codecs,
mocks and tests.
### [pkg/p2p/wire/message/payload/README.md](./pkg/p2p/wire/message/payload/README.md)

This is actually just an interface for a concurrent safe deep copy function.
### [pkg/p2p/wire/protocol/README.md](./pkg/p2p/wire/protocol/README.md)

Protocol data transport formatting and network identifiers.
### [pkg/p2p/wire/topics/README.md](./pkg/p2p/wire/topics/README.md)

Definitions of the 1 byte message topic identifiers used in the wire encoding
and by EventBus.
### [pkg/p2p/wire/util/README.md](./pkg/p2p/wire/util/README.md)

Helpers used in the pkg/p2p/wire package. In fact just has a function to get a
machine's Point of Presence on the internet, ie the gateway router of the LAN,
if its interface is not internet routable.
### [pkg/rpc/README.md](./pkg/rpc/README.md)

Contains a token based session authentication system.
### [pkg/rpc/client/README.md](./pkg/rpc/client/README.md)

Client interfaces for all of the gRPC based RPC and IPC interfaces used to
access and between Dusk blockchain and Rusk VM state machine.
### [pkg/rpc/server/README.md](./pkg/rpc/server/README.md)

Core gRPC server functionality with authentication and session management for
the clients in the client folder above.
### [pkg/util/README.md](./pkg/util/README.md)

Various utility libraries.
### [pkg/util/container/README.md](./pkg/util/container/README.md)

A ring buffer that works with raw wire/storage encoded raw bytes data. Used in
the EventBus.
### [pkg/util/container/ring/README.md](./pkg/util/container/ring/README.md)

Blurb about what this package does in one sentence here. Optional badges and
multiple paragraphs but it may belong in an Introduction if it is long.
### [pkg/util/diagnostics/README.md](./pkg/util/diagnostics/README.md)

Instrumentation for using profiling with Dusk.
### [pkg/util/legacy/README.md](./pkg/util/legacy/README.md)

Converter that upgrades the old provisioners data structure to the new Rusk
Committee.
### [pkg/util/nativeutils/README.md](./pkg/util/nativeutils/README.md)

This is a large number of libraries from the pub/sub broker eventbus, logging,
error handling, a transmission protocol, two different set implementations.
### [pkg/util/nativeutils/eventbus/README.md](./pkg/util/nativeutils/eventbus/README.md)

This is a generic interface for interconnecting different components of Dusk
including separated runtimes like the Rusk VM.
### [pkg/util/nativeutils/hashset/README.md](./pkg/util/nativeutils/hashset/README.md)

A concurrent safe hash set implementation.
### [pkg/util/nativeutils/logging/README.md](./pkg/util/nativeutils/logging/README.md)

Simplified setup and configuration for logrus.
### [pkg/util/nativeutils/prerror/README.md](./pkg/util/nativeutils/prerror/README.md)

Error type with priorities.
### [pkg/util/nativeutils/rcudp/README.md](./pkg/util/nativeutils/rcudp/README.md)

Simple short-message (max one UDP packet) reliable transport protocol using
Raptor FEC codes for retransmit-avoidance.
### [pkg/util/nativeutils/rpcbus/README.md](./pkg/util/nativeutils/rpcbus/README.md)

Process-internal message passing system based on sending return channels in
requests.
### [pkg/util/nativeutils/sortedset/README.md](./pkg/util/nativeutils/sortedset/README.md)

Binary tree (which?) based sorted set for large numbers (such as hashes).
### [pkg/util/ruskmock/README.md](./pkg/util/ruskmock/README.md)

This package contains a stand-in RUSK server, which can be used for integration
testing and DevNet launches, and replaces the Rust version of RUSK seamlessly,
thanks to the abstraction offered by Protocol Buffers and GRPC.
### [scripts/README.md](./scripts/README.md)

Miscellaneous scripts such as a Go-based build script for Dusk and some launch
control scripts.
### [scripts/docindex/README.md](./scripts/docindex/README.md)

With all packages and directories in the repository having a consistently
structured header and contents page position, it is simple to then walk the
directory tree and concatenate this section with generated rubrics that
hyperlink to the package folder where the readme with the header is found.
### [scripts/md2godoc/README.md](./scripts/md2godoc/README.md)

Automatically create godoc package headers from README.md's
