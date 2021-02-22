# SBA\* Consensus

## Intro

SBA is the first implementation of a novel consensus algorithm called Proof-Of-Blind-Bid, a privacy oriented, hybrid Proof-of-Stake like protocol with instant, statistical finality guarantees. The protocol is based on an Honest Majority of Money \(an Adversary can corrupt nodes controlling up to `f` percent of the total stake value `≥ 3f + 1`\) and weak network synchrony assumptions.

The roles in the protocol are split between two different types: Block Generators and Provisioners. Block Generators retain their privacy, with the proofs of stake computed in zero-knowledge to preserve the anonymity of the Block Generator. On the other hand, Provisioners are required to deanonymize their stakes and remain transparent about their activities in the consensus while their stake remains valid.

Both bids and stakes have a registration period of `t`, which begins when a Bid or Stake transaction is included in a final block and is required to elapse before the full-node is eligible to participate in the consensus.

SBA protocol can be conceptually defined with an inner loop \(`Block Loop`\). `Block Loop` is responsible for reaching a consensus on a uniform block.

### Security Model

The security model of SBA is based on _Snow White_, a provably secure Proof-of-Stake protocol. SBA is secure under a `∆-delayed` mildly adaptive adversary \(an adversary who is required to choose the nodes controlling a maximum of f percent of the total stake he/she is willing to corrupt ∆ rounds before the actual corruption\) and in a weakly synchronous network with a propagation delay of up to `δ` seconds.

### Block Generator

Block Generator is the first of the two full-node types eligible to participate in the consensus. To become a Block Generator, a full-node has to submit a `Bid Transaction`. The Block Generator is eligible to participate in one phase:

* `Block Generation`

In the aforementioned phase, Block Generators participate in a non-interactive lottery to be able to forge a candidate block.

### Provisioner

Provisioner is the second of the two full-node types eligible to participate in the consensus.

Unlike a Block Generator, a Provisioner node is required to deanonymize the value of the stake to be able to participate in the consensus. While it is technically possible to obfuscate the stake value, the team has decided against the latter as the addition of stake value obfuscation would have slowed down the consensus, and simultaneously increased the block size.

The Provisioner is eligible to participate in two phases:

* `Block Reduction`
* `Block Agreement`

The two mentioned phases are the way for Provisioners to reach consensus on a single block, which can then be added to the chain.

