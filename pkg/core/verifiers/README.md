# [pkg/core/verifiers](./pkg/core/verifiers)

Verification functions for various sanity checks on blocks, coinbases and
consensus messages.

<!-- ToC start -->

## Contents

section will be filled in here by markdown-toc

<!-- ToC end -->
## Stateless verification functionality

This package exposes functionality to do stateless checks on new blocks. With stateless checks, we mean anything that is not related to VM execution of the contained transactions.

### Exposed functionality

What follows is an outline of each check exposed by this package.

#### CheckBlockCertificate

Checks the block's certificate for correctness. Since the block certificate contains many fields, there is a list of checks performed by this function:

- BLS signature verification for the signatures of the first and second step
- Committee inclusion check for the included voter bitmaps for the first and second step
- Ensuring that the step number, included in the certificate, corresponds with the provided signatures and bitmaps

It should be noted, that the certificate checks are not performed on the genesis block. Due to the impossibility of the genesis block having an active committee (unless something was completely hardcoded), this block will not be checked. This does not pose a security risk, as the genesis block itself is also hardcoded.

#### CheckBlockHeader

This function performs a multitude of checks on the correctness of a block header. These checks include:

- Making sure the block header version is up-to-date with the latest one
- Ensuring that the block header contains the correct **previous block hash**, as determined by its height
- Ensuring that the block height is correct (logically follows the previous block)
- Checking that the timestamp is not before the previous block timestamp
- Calculating the transaction merkle root hash, and comparing it to the one on the block for equality

#### CheckMultiCoinbases

Simply iterates over the transactions in the block, and makes sure there is only one transaction that has the `Distribute` transaction type. Note that this function does not check whether or not the `Distribute` transaction is in the right place.

<!-- 
# to regenerate this file's table of contents:
markdown-toc README.md --replace --skip-headers 2 --inline --header "##  Contents"
-->

---
Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
