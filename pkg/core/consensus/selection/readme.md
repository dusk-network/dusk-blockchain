## Selection

`Block Generators` (a role taken outside of the Provisioner) participate in a non-interactive lottery to be able to forge a candidate block. In order to decide which Block to select among the various candidates, the `Block Generators` propagate a score associated with their candidate block, which is the result of the lottery run in a non-interactive fashion. Together with the score, the `Block Generators` publish a _zero-knowledge proof_ of correctness of the score computation according to the rules outlined in the `Blind Bid` algorithm (LINK HERE).

The `Score Selector` is the component appointed to collect the scores, verify the _zero-knowledge proof_ thereto associated and to propagate the _block hash_ associated with the highest score observed during a period of time denoted as `timeLength`. The _block hash_ is then forwarded to the `EventBus` to be picked up by the `Block Reducer`

### Values

#### Score Event

| Field | Type |
|-------|------|
| round | uint64 |
| score | uint256 |
| proof | proof |
| identity hash | uint256 |
| bid list | []byte (variable size) |
| seed | BLS signature |
| candidate block hash | uint256 |

### API

- Launch(eventbus, duration) - creates and launches the selection component, whose responsibility is to validate and select the best score among the blind bidders. The component publishes under the topic `BestScoreTopic`, once a score is chosen.

### Architecture

The `Score Selector` component follows the event driven paradigm. It is connected to the node's `EventBus` through a generic `EventFilter` and it delegates event-specific operations to its `EventHandler`.

#### Score Selection Diagram

![](docs/Score%20Selection.jpg)
