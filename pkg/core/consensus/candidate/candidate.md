# Candidate Generator

## Abstract

For more information on block generation, please refer to the [generation package readme](../generation/generation.md).

## Values

### Candidate \(Block\)

| Field | Type |
| :--- | :--- |
| Version | uint8 |
| Height | uint64 |
| Timestamp | int64 |
| Previous block hash | uint256 |
| Block seed | BLS Signature |
| Merkle root | \[\]byte \(32 bytes\) |
| Certificate | Certificate\* |
| Block hash | \[\]byte \(32 bytes\) |
| Transactions | \[\]Transaction\* \(variable size\) |

<<<<<<< HEAD:pkg/core/consensus/blockgenerator/candidate/readme.md
### Architecture
=======
\* For schemas of these items, please refer to [the dusk-wallet/v2 repo](https://github.com/dusk-network/dusk-wallet/v2).

## Architecture
>>>>>>> master:pkg/core/consensus/candidate/candidate.md

The candidate generator component is triggered by the score generator, through the `ScoreEvent` message. After receiving this message, the candidate generator will start constructing a candidate block. First off, it requests all verified transactions from the mempool. It prepends a coinbase transaction to this set, which rewards him with the agreed-upon reward amount. It then constructs the block header, and calculates the block merkle root, and the hash. Finally, it wraps the `ScoreEvent` into a `Score` message, and the newly created block into a `Candidate` message, before propagating both of these messages to the rest of the network.

