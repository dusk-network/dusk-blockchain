# [pkg/p2p/peer](./pkg/p2p/peer)

Peer to peer message sending and receiving, including one-to-many gossip message
distribution.

<!-- ToC start -->

## Contents

section will be filled in here by markdown-toc

<!-- ToC end -->

## Gotchas

### Message processing

It is required that a message which comes from the wire implements
the `payload.Safe` interface and has an unmarshalling function that can be
called through `message.Unmarshal`. Otherwise, the message will be decoded as
nil, and more often than not cause a panic.

<!-- 
# to regenerate this file's table of contents:
markdown-toc README.md --replace --skip-headers 2 --inline --header "##  Contents"
-->

---
Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
