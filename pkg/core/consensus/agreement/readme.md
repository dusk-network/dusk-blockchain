## Agreement

During the conduction of a technical analysis of sortition based consensus algorithms, the team has discovered a vulnerability, which increases the probability of a the consensus forking, dubbed a _timeout fork_. As a result, the team has concluded that SBA requires an additional step in the inner loop to guarantee statistical finality under the basic assumptions of the protocol.

`Block Agreement` is an asynchronous algorithm running in parallel with the inner loop. Successful termination of the algorithm indicates that the relevant inner loop has been successfully executed and the protocol can proceed to the next loop. The algorithm provides a statistical guarantee that at least one honest node has received a set of votes exceeding the minimum threshold required to successfully terminate the respective phase of the protocol.

### Values

#### Block Agreement Event

| Field                 | Type    |
| --------------------- | ------- |
| round                 | uint64  |
| step                  | uint64  |
| blockhash             | uint256 |
| blockhashaggregatesig | uint256 |
| publickeyset          | uint64  |

### API

- Launch(eventbus, committee, keys, round) - Launches a Block Agreement broker

### Architecture

The `Block Agreement` component follows the event driven paradigm. It is connected to the node's `EventBus` through a `broker`, and it delegates event-specific operations to its `EventHandler` implementation.

Like all the other consensus components, collection of the events and their marshalling/unmarshalling is delegated to an `EventFilter`.

#### Block Agreement Diagram

![](docs/Block%20Agreement.jpg)
