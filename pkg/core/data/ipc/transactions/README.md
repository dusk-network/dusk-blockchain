# [pkg/core/data/ipc/transactions](./pkg/core/data/ipc/transactions)

Codecs and accessors for all types of transaction messages used between Node and
Rusk VM.

<!-- ToC start -->

## Contents

1. [Gotchas](#gotchas)

<!-- ToC end -->

## Gotchas

- It is important to note, that currently, the hashing functions for
  transactions are just marshalling the entire thing, and then hashing those
  bytes. In the future, we need to make sure this aligns with how Rusk will hash
  these transactions, as the mismatch could cause potential problems down the
  line.

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)