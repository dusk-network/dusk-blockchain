## SBA\* Consensus

### Intro

SBA is the first implementation of a novel consensus algorithm called Proof-Of-Blind-Bid, a privacy oriented, hybrid Proof-of-Stake like protocol with instant, statistical finality guarantees. The protocol is based on an Honest Majority of Money (an Adversary can corrupt nodes controlling up to `f` percent of the total stake value `≥ 3f + 1`) and weak network synchrony assumptions.

The roles in the protocol are split between two different types: Block Generators and Provisioners. Block Generators retain their privacy, with the proofs of stake computed in zero-knowledge to preserve the anonymity of the Block Generator. On the other hand, Provisioners are required to deanonymize their stakes and remain transparent about their activities in the consensus while their stake remains valid.

Both bids and stakes have a registration period of `t`, which begins when a Bid or Stake transaction is included in a final block and is required to elapse before the full-node is eligible to participate in the consensus.

SBA protocol can be conceptually defined with an inner loop (`Block Loop`). `Block Loop` is responsible for reaching a consensus on a uniform block.

#### Security Model

The security model of SBA is based on _Snow White_, a provably secure Proof-of-Stake protocol. SBA is secure under a `∆-delayed` mildly adaptive adversary (an adversary who is required to choose the nodes controlling a maximum of f percent of the total stake he/she is willing to corrupt ∆ rounds before the actual corruption) and in a weakly synchronous network with a propagation delay of up to `δ` seconds.

#### Block Generator

Block Generator is the first of the two full-node types eligible to participate in the consensus. To become a Block Generator, a full-node has to submit a `Bid Transaction`.
The Block Generator is eligible to participate in one phase:

- `Block Generation`

In the aforementioned phase, Block Generators participate in a non-interactive lottery to be able to forge a candidate block.

#### Provisioner

Provisioner is the second of the two full-node types eligible to participate in the consensus.

Unlike a Block Generator, a Provisioner node is required to deanonymize the value of the stake to be able to participate in the consensus. While it is technically possible to obfuscate the stake value, the team has decided against the latter as the addition of stake value obfuscation would have slowed down the consensus, and simultaneously increased the block size.

The Provisioner is eligible to participate in two phases:

- `Block Reduction`
- `Block Agreement`

The two mentioned phases are the way for Provisioners to reach consensus on a single block, which can then be added to the chain.

### Event Driven Architecture

The consensus architecture is _event-driven_, chosen for its characteristics of allowing highly scalable applications while keeping the various architectural components highly decoupled and single-purposed. The various phases of the consensus are processed independently by single _components_, each _subscribing_ to and _publishing_ a variety of different _events_ and payloads through multiple _Topic Channels_, implemented through the `EventBus` struct.

#### Broker Topology

According to the _Broker Topology Strategy_, each component is organized as a lightweight `broker`, where the event flow gets distributed across various sub-components in a chain-like fashion. This topology was chosen in order to keep the event processing flow relatively simple and to promote reusability of the different data structures and interfaces. The broker topology has the added benefit of preventing central event orchestration.

In our implementation of the broker topology, there are three main types of recurring architectural components:

    - `Broker`: a federated struct that contains all the event channels used within the flow and which responsibility includes the creation and coordination of an `EventFilter`
    - `EventFilter`: an entity receiving the _events_ and appointed to their validation, aggregation and processing
    - one or more `channels` whereto the result of the processing get published

Additionally, several other entities are utilized within the `EventFilter` to help with code reuse and interface programming:

    - `EventHandler`: This entity contains the logic specific to the components and that cannot be shared
    - `EventQueue`: A temporary storage unit to collect messages referring to a future stage. This is possible given the asynchrony of the network which could result in nodes falling a bit behind in the event processing.
    - `Processor`: An interface component which receives filtered messages from the `EventFilter`. It conducts additional checks, and stores the received event. The `Processor` only exposes one function, `Process`, which makes it simple to create multiple implementations which fit into the `EventFilter`.

Within the codebase, two implementations of the `Processor` currently exist:

    - `Accumulator`: This component conducts additional checks on the received event, and then stores it in an `AccumulatorStore`, under an identifier which is extracted by the `EventHandler`. Once a set of events under one single identifier grows large enough, it is sent over a channel, and the `Accumulator` will clear itself for re-use.
    - `Selector`: A component which verifies an incoming event, and only stores the 'best' event it has seen. The selector can return its current 'best' event at any point, after which it is cleared.

### Common structures and interfaces

The consensus package exposes the function to create and initialize the most common channels reused across the whole package. The functions are defined as follows:

    - InitRoundUpdate(eventbus) (chan uint64) returns a channel whereto round updates get published
    - InitBlockRegeneratorCollector(eventbus) (chan AsyncState) returns a channel whose messages are interpreted as the `BLOCK_REGENERATION` signal, as outlined in the SBA* documentation.
    - InitBidListUpdate(eventBus) (chan user.BidList) returns a channel on which BidLists are sent. They contain a collection of all X values, belonging to the blind bidders in the network. Any time this BidList is updated, it is propagated to all components which use it.
