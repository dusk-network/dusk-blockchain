# Consensus

This package implements all needed components to run the [SBA\* consensus algorithm](./consensus.md) in isolation. This means, that all the "pieces" of the consensus are housed within this package, including communication infrastructure and testing frameworks. However, it still needs to be put together in order to have a fully running consensus. This code is defined in the [loop package](../loop/README.md) in this repo.

## Values

### Message Header

| Field | Type |
| :--- | :--- |
| pubkeyBLS | BLS Signature |
| round | uint64 |
| step | uint64 |
| blockhash | uint256 |

