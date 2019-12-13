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

### Values

#### Message Header

| Field     | Type          |
| --------- | ------------- |
| pubkeyBLS | BLS Signature |
| round     | uint64        |
| step      | uint64        |
| blockhash | uint256       |

### Redux-like architecture

The architecture for the implementation is based on a **Flux** or **Redux** like approach, where a single component holds the right to mutate the state, and any peripheral components/processes merely receive read-only copies of it. These peripheral components can request updates to the state, but the central `Coordinator` ultimately decides whether or not this state change will be enacted.

This `Coordinator` is accompanied by a `roundStore` which facilitates the dispatching of incoming events to other consensus components. As the `Coordinator` holds the single source of truth regarding consensus state, it redirects incoming events to the correct endpoint depending on their own state (included in the message header). The consensus components are then given the correct events, who apply further, more specific processing.

#### Event streaming

The `Coordinator` and `roundStore` are abstracted through the `EventPlayer` interface, exposing three methods:

- `Forward(id)`
- `Play(id)`
- `Pause(id)`

Each method takes in a specific identifier, which should correspond to one of the `Listeners` saved on the `roundStore`. This ensures that requests are made by components which are considered valid.

`Forward` is a way for a component to request a state change, namely incrementing the step. If the given ID is found in the `roundStore`, the step is updated and returned to the caller. `Play` and `Pause` are used to enable or disable receiving of events from the `Coordinator`. This is mainly used for directing incoming events to the correct component in the case of multiple listeners, and for disconnecting consensus components when a round is finalized.

#### Event propagation

Since the `Coordinator` has full knowledge of the state, it is also responsible for signing outgoing messages. This functionality is abstracted through the `Signer` interface and exposes two methods:

- `Gossip(topic, hash, payload, id)`
- `SendInternally(topic, hash, payload, id)`

Both methods take an id, which allows the `Coordinator` to refuse requests for sending messages from obsolete components. `Gossip` is intended for propagation to the network, while `SendInternally` is intended for internal propagation.
