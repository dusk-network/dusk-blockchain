# Synchronizer

## Responsibility

* When a block is received, the Synchronizer determines whether this block should be discarded, accepted, or queued.
* If the Synchronizer recognizes that we are behind, it will communicate with the network to ensure that all missing blocks are received.

## API

* ProcessBlock\(block\) - Determines a block's height compared to the Chain tip, and takes corresponding action.
* GetSyncProgress - (GRPC) Returns the progress of synchronization as a percentage value.

## Design

The Synchronizer is directly connected to the `peer.MessageProcessor` by registering its `ProcessBlock` callback. Any message with the topic `Block` will be sent down to the Synchronizer for further processing.

The Synchronizer will then either:
- Discard the block (if it is from the past; block height <= chain tip height)
- Send the block to the chain (block height == chain tip height + 1)
- Store the block (block height > chain tip height + 1)

It will be aware when the node is syncing or not. If the node is not syncing, the blocks which are of the correct height will be sent to the chain via the `ProcessSuccessiveBlock` callback, which passes the block through a goroutine that's responsible for consensus execution, in order to ensure successful teardown of the consensus loop. If the node is syncing, the block will be sent via the `ProcessSyncBlock` callback, which will directly go to the `chain.AcceptBlock` procedure.

Depending on whether or not the node is syncing, the Synchronizer can also request blocks from the network. This can be done in quantities of up to 500. Blocks are requested by gossiping a `GetBlocks` message, using the chain tip as the locator hash, which informs nodes about where we are in the chain.
