# [pkg/core/consensus/user](./pkg/core/consensus/user) - Provisioners and Sortition

This package implements the data structure which holds the Provisioner
committee, and implements methods on top of this committee in order to be able
to extract **voting committees** which are eligible to decide on blocks during
the SBA\* consensus protocol.

<!-- ToC start -->
##  Contents

   1. [Abstract](#abstract)
   1. [Gotchas](#gotchas)
<!-- ToC end -->

## Abstract

Deterministic sortition is an optimization of cryptographic sortition introduced
by Micali et al. It extends the functionality of cryptographic sortition in a
Random Oracle setting in a non-interactive fashion, improving both the network
throughput and space-efficiency.

Deterministic sortition is an algorithm that recursively hashes the public seed
with situational parameters of each step, mapping the outcome to the current
stakes of the Provisioners in order to extract a pseudo-random `Committee`, per
step.

## Gotchas

- When wrapping around the provisioner set in the sortition loop, it is of
  absolute importance to ensure that the index is reset to 0, to fully reset the
  loop. Before, this index was never reset, and after traversing the provisioner
  set once, the first provisioner was the only one who got chosen time after
  time until the loop finished. This could, in certain scenarios, pose a
  potential attack vector for a node to be able to achieve a majority vote and
  therefore, break consensus rules.

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
