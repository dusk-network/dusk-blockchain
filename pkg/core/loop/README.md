# # [pkg/core/loop](./pkg/core/loop)

The `Loop` is the most high-level component of the SBA\* consensus
implementation. It simulates a state-machine like architecture, with a
functional flavor, where consensus phases are sequenced one after the other,
returning functions as they complete, based on their outcomes.

<!-- ToC start -->

## Contents

1. [Architecture](#architecture)
    1. [Communications](#communications)
    1. [Running](#running)

<!-- ToC end -->

## Architecture

For any caller to be able to start the consensus, they will need to have access
to a fully initialized `Loop`. It is initialized by giving a couple of
pre-requisite items:

- A public key, to which block generator rewards can be attributed
- An [`Emitter`](#communications), responsible for routing communications
- A `Requestor`, which is capable of retrieving candidate blocks on request from
  the network
- A callback to be used for candidate block verification
- A database instance
- The initial timeout

This `Loop` can then be re-used for the lifetime of the node to start and
maintain consensus loops.

### Communications

The `Loop` is responsible for establishing the communications bridge between the
consensus and the rest of the node. This is done by creating
an [`Emitter`](../consensus/comms.go) and passing it to the `Loop` on startup.
The `Loop` will then set up message channels for each of the messages we expect
to receive for the consensus, and subscribe them to the event bus with the
relevant topics. From this point, all consensus messages will be directly routed
to the correct components as they start coming in.

### Running

To start the consensus loop, simply create the state machine by
calling `CreateStateMachine` on the `Loop` instance. This will return two
consensus phases - the score phase, and the agreement phase. There is a key
distinction between the two, which is that the score phase will return another
function as it completes, and so will it's successors until they reach the score
phase again. This ensures the sequential, state-machine-like progression of the
synchronous consensus phases. The agreement phase is intended to run
concurrently, as it can receive messages at any time, and is not bound to a
timer or anything of the sort.

With these two phases, all we have left to do to start the consensus loop, is to
formulate a [`RoundUpdate`](../consensus/comms.go#L50). This contains all the
stateful information needed by the consensus to do its job. Finally, with all of
these items in place, call `loop.Spin`, passing these items, in order to launch
the consensus loop. Once this is called, the consensus will progress until an
error is encountered, or until it is cancelled through a context cancellation.

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
