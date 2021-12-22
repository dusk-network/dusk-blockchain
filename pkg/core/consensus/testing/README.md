# [pkg/core/consensus/testing](./pkg/core/consensus/testing)

Miscellaneous functions used in testing the consensus.

<!-- ToC start -->

## Contents

section will be filled in here by markdown-toc

<!-- ToC end -->

## Consensus integration testbed

This package contains an integration testing suite designed to test interactions
between the [chain](../../chain/README.md) component, and
the [consensus](../README.md). It is specifically designed to remove as many
peripherals as possible in order to fully isolate and properly test these two
components, so that we can easily catch race conditions, corner cases and bugs
alike.

### How it works

The integrating testing suite entirely bypasses the network layer, and thus does
not use anything in the `peer` package. Instead, all nodes which participate
will share a single event bus, which facilitates all communication. In order to
re-propagate `topics.Gossip` messages to other peers, a small utility has been
introduced - the `gossipRouter`, which catches any gossiped messages, decodes
them, and publishes them on the event bus. This way, all communications are
locally contained, and subject to minimal latency and ambiguity of delivery.

![Diagram of the testbed architecture with 2 nodes](./consensus_testbed_setup.jpg)

As you can see, the consensus is still ran separately for each node, however,
the communication channels are shared to facilitate extra simplicity to the
integration testing suite. A node in this testing suite simply consists of
a `chain.Chain`, which holds a `loop.Consensus`. Therefore, we are fully able to
test the interaction of these two components. The `chain.Chain` will update the
caller of the progress in the consensus, by using the `AcceptedBlock` topic.

### How to use

Currently, there are only two configurable aspects of the integration testing
suite:

- The amount of nodes
- The amount of rounds it is supposed to run for

Since the only test is currently purely to see if nodes can reach a certain
point in the consensus, there is not much extra configurability.

To configure the amount of nodes, simply set the `DUSK_TESTBED_NUM_NODES`
environment variable. To configure the amount of rounds, set
the `DUSK_TESTBED_NUM_ROUNDS` environment variable.

```bash
$ export DUSK_TESTBED_NUM_NODES=10
$ export DUSK_TESTBED_NUM_ROUNDS=10
```

### Future work

This integration testing suite is currently quite simplistic, but should be
easily extendable and updatable in the future, once the need for more test cases
arises.

<!-- 
# to regenerate this file's table of contents:
markdown-toc README.md --replace --skip-headers 2 --inline --header "##  Contents"
-->

---
Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
