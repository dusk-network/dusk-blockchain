## Consensus Notations
### Block
A set of data.
### Block Header
A section of the block containing the metadata.
### Block Body
A section of the block containing the transactions.
### Block Certificate
A section of the block containing the Provisioner signatures of a block hash, validating the block
### Block Generator
A node that is eligible to participate in the Block Generation phase after submitting a Blind Bid.
### Provisioner
A node that is eligible to participate in the Block Reduction, Binary Agreement and Set Reduction phases after submitting a stake.
### Blind Bid
A transaction type that is used to enroll a node as a Block Generator, while obfuscating the identity and the amount of DUSK tokens staked.
### Stake
A transaction type that is used to enroll a node as a Provisioner, while revealing the identity and the amount of DUSK tokens staked.
### Seed
A pseudo-random integer attached to the block header, created by the Block Generator signing a seed of a previous block, in case of a candidate block, or created by hashing seed of a previous block in case of an empty block.
### Round
A block height.
### Step
The iterative counter of the consensus phases.
### Candidate Block
A block created by the Block Generator, which can be elected in the round.
### Block Generation
The first phase of the protocol, responsible for the generation of candidate blocks.
### Block Reduction
The second phase of the protocol, responsible for an agreement on a single candidate block, in case of a discrepancy in the initial values (the candidate blocks with the highest score) or defaulting to an empty block in case an agreement cannot be reached.
### Binary Agreement
The third phase of the protocol, responsible for electing the block to be appended to the blockchain.
### Set Reduction
The fourth phase of the protocol, executed in case of a candidate block being elected during the Binary Agreement phase, responsible for an agreement on common set of Provisioner signatures to be added to the Block Certificate.
### Sortition
A lottery responsible for probabilistically electing candidates to participate in the consensus phases.
### Provisioner Extraction
An outcome of a sortition lottery applied during the Block Reduction, Binary Agreement and Set Reduction to elect a subset of the Provisioner set.