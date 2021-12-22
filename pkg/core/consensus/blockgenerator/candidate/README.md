# Candidate Generator Component

This package implements a candidate generator, for the blind-bid protocol of the SBA\* consensus protocol.

<!-- ToC start -->

## Contents

section will be filled in here by markdown-toc

<!-- ToC end -->

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

## Architecture

The candidate generator exposes its functionality through a single interface method: `GenerateCandidateMessage`. It should logically follow a call to the score generator to generate a score proposal. This score proposal should also be fed into the candidate generator, which combines it with the candidate block, in order to create a fully-fledged `Score` message.

The component should be initialized with a public key, to which the potential reward can be attributed. Once called, the block generator goes through these steps:

- It will generate a committee for the given round and step. This committee contains all the people that can potentially be rewarded, if this block is finalized
- The generator will ask the mempool for a list of transactions, up to a certain size (determined by the block size cap)
- A coinbase transaction will be appended to the end of the list, as per the consensus rules
- A block header is constructed, leaving only the certificate field empty. This certificate is constructed later, in the agreement phase
- The block is put together, and is then concatenated with the score proposal, to create a `Score message. This message is then returned to the caller

This message is then fully ready to be encoded and gossiped to the network.

<!-- 
# to regenerate this file's table of contents:
markdown-toc README.md --replace --skip-headers 2 --inline --header "##  Contents"
-->

---
Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)