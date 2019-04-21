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

    - `Broker`: a federated struct that contains all the event channels used within the flow and which responsibility includes the creation and coordination of a `Collector`
    - `Collector`: an entity receiving the _events_ and appointed to their validation, aggregation and processing
    - one or more `channels` whereto the result of the processing get published

Additionally, several other entities are utilized within the `Collector` to help with code reuse and interface programming:

    - `EventHandler`: This entity contains the logic specific to the components and that cannot be shared
    - `EventQueue`: A temporary storage unit to collect messages referring to a future stage. This is possible given the asynchrony of the network which could result in nodes falling a bit behind in the event processing.
    - `AccumulatorStore`: A map that stores consensus messages. This can be used to track message accumulation for specific keys, such as a step, or a particular block hash.

### Common structures and interfaces

The consensus package exposes the function to create and initialize the most common channel reused across the whole package. This common channel is for receiving notifications about Rounds.

    - InitRoundUpdate(eventbus) (chan uint64) returns a channel whereto round updates get published

Additionally, the consensus package exposes a marshaller and unmarshaller for consensus message headers, as well as the `EventHeader` struct itself. These data structures and methods are common part of nearly all other consensus sub-packages, and help to reduce code duplication.

    - `EventHeader`: This struct contains all the common fields included in all consensus messages, except for the `Score` message.
    - `EventHeaderMarshaller`: A struct which contains the logic to turn an `EventHeader` into a byte stream representation of itself.
    - `EventHeaderUnmarshaller`: A struct which contains the logic to turn the byte stream representation of an `EventHeader` into it's struct form.
