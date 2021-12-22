# Chain component

The `Chain` is a component which is fully responsible for building the blockchain. It judges the validity of incoming blocks, appends them to the database, and disseminates newly approved blocks both internally (through the event bus) and externally (over the gossip network).

Additionally, it is aware of the node's state in the network at all times, and controls the execution of the consensus by responding to perceived state changes.

<!-- ToC start -->

## Contents

section will be filled in here by markdown-toc

<!-- ToC end -->

## Structure

To the rest of the system, the `Chain` appears as quite a monolithic component. However, it is composed by a few different, more granular components, which all fulfill their own responsibilities.

### Loader

The `Loader` is responsible for initializing the `Chain` with the correct initial data, as well as checking for data integrity on startup. It abstracts over the DB, and is implemented by [the loader](./loader.go).

### Verifier

The `Verifier` abstracts over block verification mechanisms, and uses the [`verifiers` package](../verifiers/README.md) under the hood, in an attempt to simplify the API (as checks are supposed to be kept atomic from a function perspective, but always are called together). It is implemented by [the loader](./loader.go).

### Synchronizer

The `Synchronizer` is a stand-alone component which is fully responsible for processing blocks that come in from the network. Through the `Ledger` interface, it can direct the `Chain` into the right decisions based on the perceived state of the node against the network. More information can be found in the [synchronizer documentation](./synchronizer.md).

Below follows a visual workflow of how the chain and synchronizer take decisions on incoming blocks, which should help clarify the control flow.

![Block processing decision tree](./chain_processing_flow.jpg)

### Loop

The `Loop` allows the `Chain` to take control of consensus execution, by allowing it to easily start and stop the work being done.

### Proxy

The `Proxy` is used to contact the RUSK server. The `Chain` outsources certain operations to RUSK since the current codebase can not execute VM transactions, and this is needed for transaction validity checks and state transitions.

## Synchronizer

### Responsibility

* When a block is received, the Synchronizer determines whether this block should be discarded, accepted, or queued.
* If the Synchronizer recognizes that we are behind, it will communicate with the network to ensure that all missing blocks are received.

### API

* ProcessBlock\(block\) - Determines a block's height compared to the Chain tip, and takes corresponding action.
* GetSyncProgress - (GRPC) Returns the progress of synchronization as a percentage value.

### Design

The Synchronizer is directly connected to the `peer.MessageProcessor` by registering its `ProcessBlock` callback. Any message with the topic `Block` will be sent down to the Synchronizer for further processing.

The Synchronizer will then either:
- Discard the block (if it is from the past; block height <= chain tip height)
- Send the block to the chain (block height == chain tip height + 1)
- Store the block (block height > chain tip height + 1)

It will be aware when the node is syncing or not. If the node is not syncing, the blocks which are of the correct height will be sent to the chain via the `ProcessSuccessiveBlock` callback, which passes the block through a goroutine that's responsible for consensus execution, in order to ensure successful teardown of the consensus loop. If the node is syncing, the block will be sent via the `ProcessSyncBlock` callback, which will directly go to the `chain.AcceptBlock` procedure.

Depending on whether or not the node is syncing, the Synchronizer can also request blocks from the network. This can be done in quantities of up to 500. Blocks are requested by gossiping a `GetBlocks` message, using the chain tip as the locator hash, which informs nodes about where we are in the chain.


<!-- 
# to regenerate this file's table of contents:
markdown-toc README.md --replace --skip-headers 2 --inline --header "##  Contents"
-->

---

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
