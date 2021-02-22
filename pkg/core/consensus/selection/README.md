# Selection Component

The `Selector` is the component appointed to collect the scores, verify the _zero-knowledge proof_ thereto associated and to propagate the _block hash_ associated with the highest score observed during a period of time denoted as `timeLength`. The _block hash_ is then forwarded to the `EventBus` to be picked up by the `Block Reducer`

An abstract description of the Selection phase in the SBA\* consensus protocol can be found [here](./selection.md).

## Values

### Score Message

| Field | Type |
| :--- | :--- |
| score | uint256 |
| proof | proof |
| identity hash | uint256 |
| seed | BLS signature |
| previous block hash | uint256 |
| candidate block | [block.Block](../../data/block/block.go) |

## Architecture

The Selection component is implemented through the `Phase` struct. Internally, it carries a `BlockGenerator` in order to fully synchronize the creation of a candidate block and score, with the Selection phase. When the Selection component is called, the first thing it does is this block generation routine. Once finished (or if no block generator was actually initialized), queued events are flushed in order to catch up with any delays, and a listening loop is started. The selection component will listen on four channels:

- The `internalScoreResult` chan, on which it can receive a `Score` message from its own block generator - this message is then forwarded into the standard processing pipeline
- The `evChan`, which catches events coming from the network
- The `timeOutChan`, responsible for telling the Selection component when the time for the selection phase has ran out, and the component needs to wrap up its activites
- The `ctx.Done` chan, which notifies the Selection component of context cancellations, and allows for swift cleanup of the hanging goroutine

When an event comes through over the `evChan`, the following processing pipeline is followed:

1. Check if the event score is higher than the current `bestEvent`. If not, the flow ends here.
2. Verify the proof, included in the `Score` event. If the verification fails, the flow ends here.
3. The `Score` event is repropagated to the network.
4. The `bestEvent` field on the Selection component will now be set to the new `Score` event.

The purpose of this processing flow is to have the `Score` message with the highest score in memory when the timer expires. Once this timer expires, the message is returned and can be used to vote in the [reduction step](../reduction/README.md).
