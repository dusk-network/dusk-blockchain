# Selection

## Abstract

`Block Generators` \(a role taken outside of the Provisioner\) participate in a non-interactive lottery to be able to forge a candidate block. In order to decide which Block to select among the various candidates, the `Block Generators` propagate a score associated with their candidate block, which is the result of the lottery run in a non-interactive fashion. Together with the score, the `Block Generators` publish a _zero-knowledge proof_ of correctness of the score computation according to the rules outlined in the `Blind Bid` algorithm \(LINK HERE\).

The `Score Selector` is the component appointed to collect the scores, verify the _zero-knowledge proof_ thereto associated and to propagate the _block hash_ associated with the highest score observed during a period of time denoted as `timeLength`. The _block hash_ is then forwarded to the `EventBus` to be picked up by the `Block Reducer`

## Values

### Score Message

| Field | Type |
| :--- | :--- |
| score | uint256 |
| proof | proof |
| identity hash | uint256 |
| bid list | \[\]byte \(variable size\) |
| seed | BLS signature |
| previous block hash | uint256 |
| candidate block hash | uint256 |

## Architecture

The `Selector` is triggered by a `Generation` message, which denotes the reset of the consensus loop. Upon receiving this message, the `Selector` will start a timer. Between the start and the expiry of the timer, the `Selector` will enable streaming of `Score` events. Any time a `Score` event is received, it goes through this processing flow:

1. Check if the event score is higher than the current `bestEvent`. If not, the flow ends here.
2. Verify the proof, included in the `Score` event. If the verification fails, the flow ends here.
3. The `Score` event is repropagated to the network.
4. The `bestEvent` will now be set to the new `Score` event.

The purpose of this processing flow is to have the `Score` event with the highest score in memory when the timer expires. Once this timer expires, the selector will pause event streaming, and publish this `Score` event internally, under the `BestScore` topic.

