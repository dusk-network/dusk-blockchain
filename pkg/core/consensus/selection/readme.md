## Selection

The _selection_ package includes the following components:

- `Score Selector`
- `SigSet Generator`
- `SigSet Selector`

### Score

`Block Generators` (a role taken outside of the Provisioner) participate in a non-interactive lottery to be able to forge a candidate block. In order to decide which Block to select among the various candidates, the `Block Generators` propagate a score associated with their candidate block, which is the result of the lottery run in a non-interactive fashion. Together with the score, the `Block Generators` publish a _zero-knowledge proof_ of correctness of the score computation according to the rules outlined in the `Blind Bid` algorithm (LINK HERE).

The `Score Selector` is the component appointed to collect the scores, verify the _zero-knowledge proof_ thereto associated and to propagate the _block hash_ associated with the highest score observed during a period of time denoted as `timeLength`. The _block hash_ is then forwarded to the `EventBus` to be picked up by the `Block Reducer`

### SigSet

The `SigSet Selector` also handles the `SigSet Generation` process. The `Sigset Generation` is applied in order to form a uniform signature set of the Provisioners to attest the winning candidate _block hash_ during the Block Reduction phase. The `SigSet Selection` selects the signature set based on the highest stake among the `Committee` forming Provisioners.

### Values

#### Score Event

| Field | Type |
| opcode | uint8 |
| round | uint64 |
| step | uint64 |
| score | uint256 |
| proof | proof |
| candidateblockhash | uint256 |
| prevblockhash | uint256 |
| certificate | certificate |

#### SigSet Generation Event

| Field | Type |
| opcode | uint8 |
| round | uint64 |
| step | uint64 |
| blockhash | uint256|
| sigset | sigset|
| prevblockhash | uint256|

### API

    - LaunchScoreSelectionComponent(eventbus, duration) - creates and launches the component which responsibility is to validate and select the best score among the blind bidders. The component publishes under the topic BestScoreTopic
    - LaunchSignatureSelector(eventbus, committee, duration) - Launches a the `SigSet Selector` component which responsibility is to collect the signature set until a quorum is reached, select the one associated with the highest Provisioner stake and propagate the selection further

### Architecture

Both the `Score Selector` and the `SigSet Selector` components follow the event driven paradigm. They both are connected to the node's `EventBus` through a generic `Collector` and delegate event-specific operations to their own `EventHandler`. They both make use of a `Selector` component to help code reuse.
