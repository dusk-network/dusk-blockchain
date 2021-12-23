# [pkg/core/consensus/blockgenerator](./pkg/core/consensus/blockgenerator)

This package implements a full block generator component, to be used by
participants of the blind-bid protocol in the SBA\* consensus. It is internally
made up of two distinct components, a [score generator](./score/README.md), and
a [candidate generator](./candidate/README.md).

<!-- ToC start -->

## Contents

1. [Abstract](#abstract)
1. [Blind Bid Algorithm](#blind-bid-algorithm)
    1. [Procedure](#procedure)

<!-- ToC end -->

## Abstract

`Block Generator` is the first of the two full-node types eligible to
participate in the consensus \(the other being the `Provisioner`\). To become
a `Block Generator`, a full-node has to submit a `Bid Transaction`.
The `Block Generator` is eligible to participate in one phase - the _block
generation_ where the `Block Generators` participates in a non-interactive
lottery to be able to forge a candidate block.

## Blind Bid Algorithm

The block generation makes use of `bulletproof`-based _zero-knowledge
cryptography_ in order to prove the correct computation of a _score_ associated
with a `block candidate` which gets validated and selected by Provisioners
during the _score selection_ phase carried out by the `selection` package.

### Procedure

The `Blind Bid` algorithm is outlined in the following steps

1. A universal counter `N` is maintained for all bidding transactions in the
   lifetime.
2. Seed `S` is computed and broadcasted.
3. Bidder selects secret `K`.
4. Bidder sends a bidding transaction with data `M = H(K)`.
5. For every bidding transaction with d coins and data M an entry `X = H(d,M,N)`
   is added to `T`. Then `N` is increased.
6. Potential bidder computes `Y =H(S,X)`, score `Q=F(d,Y)`, and
   identifier `Z =H(S,M)`.
7. Bidder selects a bid root `RT` and broadcasts
   proof `π = Π(Z, RT, Q, S; K, d, N)`.

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
