## Agreement

### Abstract

During the conduction of a technical analysis of sortition based consensus algorithms, the team has discovered a vulnerability, which increases the probability of a the consensus forking, dubbed a _timeout fork_. As a result, the team has concluded that SBA requires an additional step in the inner loop to guarantee statistical finality under the basic assumptions of the protocol.

`Block Agreement` is an asynchronous algorithm running in parallel with the inner loop. Successful termination of the algorithm indicates that the relevant inner loop has been successfully executed and the protocol can proceed to the next loop. The algorithm provides a statistical guarantee that at least one honest node has received a set of votes exceeding the minimum threshold required to successfully terminate the respective phase of the protocol.

### Values

#### Block Agreement Event

| Field                          | Type          |
| ------------------------------ | ------------- |
| Aggregated public keys\*       | BLS APK       |
| Committee bit-representation\* | uint64        |
| Aggregated signatures\*        | BLS Signature |
| Signature of all fields        | BLS Signature |

\* These fields appear twice, once for each step of [Reduction](../reduction/readme.md).

### Architecture

The agreement component is a special case - the `Coordinator` has different state-based filtering rules for `Agreement` messages, since this phase runs asynchronously from all the others, and does not make use of any timers. The agreement component will listen for messages the moment it is initialized, also slightly differing from the other consensus components.

The agreement component filters incoming messages by checking their validity from a voting committee perspective, and by checking the validity of the aggregated signatures and public keys. Verified events will be sent to the `Accumulator`, which acts as a store for `Agreement` messages, sorting them by step (since these messages are the product of a two-step reduction cycle). If enough messages for a given step enter the `Accumulator` and it reaches quorum, the `Accumulator` signals the agreement component to send two messages: a `Finalize` message, which notifies the `Coordinator` to disconnect the current `roundStore` and instantiate a fresh one, and a `Certificate` message. The `Certificate` message is generated from one of the `Agreement` messages collected for the winning step, and is published internally via the `consensus.Signer`.
