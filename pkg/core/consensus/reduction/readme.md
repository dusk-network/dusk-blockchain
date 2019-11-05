## Binary Reduction

### Abstract

The Binary Reduction algorithm lays at the core of SBA\*. It converts the problem of reaching consensus on arbitrary values to reaching consensus on one of two values. It is an adaptation of the Turpin and Coan algorithm, originally concocted to solve the general Byzantine agreement when given a binary Byzantine agreement algorithm as a subroutine, for `n > 3f` (with `n` defined as total number of nodes and `f` defined as adversarial nodes).

Unlike other implementations, which normally utilize the original algorithm, Binary Reduction adopted in SBA\* follows a two-step approach, with the input of the second step depending on the output of the first one.

If no consensus have been reached on a uniform value, the algorithm returns a default value and waits for the next instantiation.

Binary Reduction acts as a uniform value extraction function which is then fed through the Block Agreement algorithm before exiting the loop in case of a successful termination of the Block Agreement algorithm.

### Values

#### Block Reduction Event

| Field           | Type          |
| --------------- | ------------- |
| signedblockhash | BLS Signature |

### Architecture

The reduction phase is split up into two components - one for each step. This is because, despite their similarity in logic, subtle nuances between the two steps make it more feasible to split the logic between two different components, for readability purposes. At the core, a `Reducer` works like this:

- It gets triggered by a message of a certain topic (first step: `BestScore` / second step: `StepVotes`). This instantiates an `Aggregator` and starts a timer.
- It starts collecting Reduction messages, passing them down to the `Accumulator`
- When the `Aggregator` reaches quorum, or when the timer is triggered, the `Reducer` will send out one or more messages (first step: `StepVotes` / second step: `Restart` & `Agreement`)
- It stops collecting events, and waits for the next `BestScore` or `StepVotes`

#### Aggregator

Each `Reducer` makes use of an `Aggregator`, which is a component akin to a storage for incoming messages. It is instantiated with a callback, which it will call after a certain amount of messages are collected. The `Aggregator` will receive any incoming Reduction messages after they are filtered by the `Coordinator` and the `Reducer`. It will separate messages by their block hash, and proceed to aggregate the included `signedblockhash` with other collected signatures for this hash (if any). Additionally, it saves the senders BLS public key in a `sortedset.Set`. Once the amount of keys and signatures for a certain blockhash exceeds a threshold, the `Aggregator` will forward the collected information, by providing it as an argument for its given callback. After triggering this callback, the `Aggregator` will no longer accept new messages, and a new instance needs to be created for subsequent steps.
