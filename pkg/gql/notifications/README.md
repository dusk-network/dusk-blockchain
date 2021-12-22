# [pkg/gql/notifications](./pkg/gql/notifications)

Notification system to broadcast updates to subscribed websocket clients. It can
be considered as a bridge between EventBus events and websocket connections. In
most cases, a subscriber would be the UI client expecting updates on new block
accepted. Additionally, this package could be extended to provide subscriptions
for more specific events \(e.g a client needs a notification on a specific
tx\_hash accepted\)

<!-- ToC start -->

## Contents

section will be filled in here by markdown-toc

<!-- ToC end -->

## Design

Server instatiates a pool of brokers. Each Broker is capable of receiving
updates from the eventBus \(e.g topics. acceptedBlock, topics.Tx, etc\). On
update occurrence, a broker broadcasts a message to its list of subscribed
clients.

## Messages

Currently, the only notification sent is intended to satisfy Block Explorer UI
needs. \(pending to revise the format of the message\)

### On block accepted

```javascript
{
    "Height"
:
    183203,
        "Hash"
:
    "abbd8544211d7ed8b1472fcae10759f6735aca3dd604937900d44bb88cdb52d6",
        "Timestamp"
:
    1579184799,
        "Txs"
:
    [
        "f09f6522cc7ad80697ca63a90507cf7bb303bd4c6517f936300842f07e6ae056"
    ],
        "BlocksGeneratedCount"
:
    58794
}
```

### Configuration

```text
[gql.notification]
# Number of pub/sub brokers to broadcast new blocks. 
# 0 brokersNum disables notifications system
brokersNum = 10
clientsPerBroker = 1000
```

### Examples

`TestWebsocketEndpoint` introduces a sample websocket client connecting
to `ws://127.0.0.1:9001/ws` and consuming notifications


<!-- 
# to regenerate this file's table of contents:
markdown-toc README.md --replace --skip-headers 2 --inline --header "##  Contents"
-->

---
Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
