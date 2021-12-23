# [pkg/core/consensus/reduction](./pkg/core/consensus/reduction)

This package defines the components for both the first and second step of
reduction individually, as specified in
the [Binary Reduction Phase](./reduction.md) of the SBA\* consensus protocol.

Due to small, but intricate differences between the two steps, the components
are defined individually, with a minimal amount of shared code. This was done in
an effort to increase readability, since it avoids massive amounts of
abstraction which, on earlier iterations of the package, caused some confusion
for readers. That said, code which is identical across the two components is
defined in the top-level of the package, and imported down.

<!-- ToC start -->
##  Contents

   1. [Abstract](#abstract)
   1. [Values](#values)
      1. [Block Reduction Event](#block-reduction-event)
   1. [Architecture](#architecture)
      1. [Differences between the two components](#differences-between-the-two-components)
      1. [Aggregator](#aggregator)
<!-- ToC end -->

## Abstract

The Binary Reduction algorithm lays at the core of SBA\*. It converts the
problem of reaching consensus on arbitrary values to reaching consensus on one
of two values. It is an adaptation of the Turpin and Coan algorithm, originally
concocted to solve the general Byzantine agreement when given a binary Byzantine
agreement algorithm as a subroutine, for `n > 3f` \(with `n` defined as total
number of nodes and `f` defined as adversarial nodes\).

Unlike other implementations, which normally utilize the original algorithm,
Binary Reduction adopted in SBA\* follows a two-step approach, with the input of
the second step depending on the output of the first one.

If no consensus have been reached on a uniform value, the algorithm returns a
default value and waits for the next instantiation.

Binary Reduction acts as a uniform value extraction function which is then fed
through the Block Agreement algorithm before exiting the loop in case of a
successful termination of the Block Agreement algorithm.

## Values

### Block Reduction Event

| Field | Type |
| :--- | :--- |
| signedblockhash | BLS Signature |

## Architecture

At the core, a `Reducer` works like this:

* It gets triggered by a call to the `Run` function. This instantiates
  an `Aggregator`, starts a timer, and gossips a `Reduction` message (using the
  provided keys on startup)
* The queue is flushed, and it starts collecting Reduction messages, passing
  them down to the `Aggregator`
* When the `Aggregator` reaches quorum, or when the timer is triggered,
  the `Reducer` will return a message
* The component is then finished and waits for the next call to `Run`

### Differences between the two components

Since there are subtle differences in the actions that need to be taken per
step, the reducers have been split up in a [first step](./firststep/)
and [second step](./secondstep/). Between the two, these differences can be
found:

- Upon reaching quorum, the first step reducer will attempt to retrieve the
  candidate block corresponding to the winning hash (either through the DB or
  the network), and attempt to verify it, to make sure it's okay to continue
  voting on this block for the second step
- When reaching quorum, the first step reducer will **return** a `StepVotes`
  message, which is passed on to the second step reducer. The second step
  reducer will instead **gossip** an `Agreement` message, using the
  combined `StepVotes` of the first and second step to create a certificate. The
  second step reducer does not return anything

### Aggregator

Each `Reducer` makes use of an `Aggregator`, which is a component akin to a
storage for incoming messages. The `Aggregator` will receive any incoming
Reduction messages after they are filtered by the `Reducer`. It will separate
messages by their block hash, and proceed to aggregate the
included `signedblockhash` with other collected signatures for this hash \(if
any\). Additionally, it saves the senders BLS public key in a `sortedset.Set`.
Once the amount of keys and signatures for a certain blockhash exceeds a
threshold, the `Aggregator` will return the collected information.

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
