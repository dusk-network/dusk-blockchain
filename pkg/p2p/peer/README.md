# [pkg/p2p/peer](./pkg/p2p/peer)

Peer to peer message sending and receiving, including one-to-many gossip message
distribution.

<!-- ToC start -->
##  Contents

   1. [Gotchas](#gotchas)
      1. [Message processing](#message-processing)
<!-- ToC end -->

## Gotchas

### Message processing

It is required that a message which comes from the wire implements
the `payload.Safe` interface and has an unmarshalling function that can be
called through `message.Unmarshal`. Otherwise, the message will be decoded as
nil, and more often than not cause a panic.

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
