## Agreement

During the conduction of a technical analysis of sortition based consensus algorithms, the team has discovered a vulnerability, which increases the probability of a the consensus forking, dubbed a _timeout fork_. As a result, the team has concluded that SBA requires an additional step in both of the inner loops to guarantee statistical finality under the basic assumptions of the protocol.

`Set Agreement` is an asynchronous algorithm running in parallel with the inner loop. The protocol includes two variation of the Set Agreement algorithm - `Block Agreement` and `Sigset Agreement`. Successful termination of one of the variations of the algorithm indicates that the relevant inner loop has been successfully executed and the protocol can proceed to the next loop. The algorithm provides a statistical guarantee that at least one honest node has received a set of votes exceeding the minimum threshold required to successfully terminate the respective phase of the protocol.

### Values

#### Block Agreement Event

| opcode | uint8 |
| round | uint64 |
| step | uint64 |
| blockhash | uint256 |
| blockhashaggregatesig | uint256 |
| publickeyset | uint64 |
| prevblockhash | uint256 |

#### SigSet Agreement Event

| Field | Type |
| opcode | uint8 |
| round | uint64 |
| step | uint64 |
| blockhash | uint256 |
| sigsethash | uint256 |
| sigsethashaggregatesig | uint256 |
| publickeyset | uint64 |
| prevblockhash | uint256 |

### API

    - LaunchBlockNotary(eventbus, committee, duration) - Launches a Block Agreement broker
    - LaunchSigSetAgreement(eventbus, committee, duration) - Launches a SigSet Agreement broker

### Architecture

Both the `Block Agreement` and the `SigSet Agreement` components follow the event driven paradigm. They both are connected to the node's `EventBus` through a generic `Agreement` and delegate event-specific operations to their own `EventHandler` implementation.

Like all the other consensus components, collection of the events and their marshalling/unmarshalling is delegated to a `Collector`
