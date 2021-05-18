 A Kadcast implementation in pure Golang
=============
 

`p2p/kadcast`  package is an attempt to implement kadcast protocol specification from https://eprint.iacr.org/2019/876.pdf. It basically includes  Kademlia routing state and Message propagation algorithm. For the purpose of message propagation, Raptor Codes  [RFC5053](https://tools.ietf.org/html/rfc5053)  implementation from [gofountain](https://github.com/google/gofountain/) is used instead of the recommended fountain code - `RaptorQ` (more details `pkg/util/nativeutils/rcudp/README.md`).

## Usage
--------------

#### Configuration
```toml

# By enabling it, node will join an experimental kadcast network
# In addition, topics.Block and topics.Tx will be propagated in kadcast network
# NB: The messages propagated in kadcast are not propagated in gossip
enabled=false

# Enable/Disable RC-UDP transport
raptor=false

# Both listeners (UDP and TCP) are binding on this local addr
# NB The addr should be reachable from outside
address="127.0.0.1:7100"

# Maximum delegates per bucket 
# System parameter β from protocol
maxDelegatesNum=3

# Example list of bootstarpping nodes
bootstrappers=["voucher.dusk.network:9090","voucher.dusk.network:9091"]

```

#### Code snippet

```golang

# Initiate kadcast message propagation
event = message.NewWithHeader(topics.Block, *buf, []byte{255})
eventBus.Publish(topics.Kadcast, event)
```

## Broadcast Message flow
--------------


1. Publish `topics.Kadcast` event with message payload and kadcast height 
2. `kadcast.Writer` handles `topics.Kadcast` event
3. `kadcast.Writer` serialize the event into `Kadcast Wire Message` of type `Broadcast`
4. kadcast.Writer performs kadcast propagation algorithm where transport protocol is
 - ` RC-UDP`, if `kadcast.raptor=true` (config)
- `TCP Dial and Send`, if `kadcast.raptor=false`

## Point-to-point Message flow
--------------

Point-to-point messaging is a protocol extension that allows any node to request data from a specified peer. An example situation is a peer requesting missing blocks from a peer on synchronization procedure. 

1. Publish `topics.KadcastPoint` event with message payload,  destination peer and kadcast height = 0, 
2. `kadcast.Writer` handles `topics.KadcastPoint` event
3. `kadcast.Writer` serialize the event into `Kadcast Wire Message` of type `Broadcast`
4. kadcast.Writer performs kadcast propagation algorithm where transport protocol is
 - ` RC-UDP`, if `kadcast.raptor=true` (config)
- `TCP Dial and Send`, if `kadcast.raptor=false`



## Transport protocols
--------------

Current version of kadcast supports two different methods of sending/receiving a message to/from a kadcast peer - `TCP Dial and Send` and `RC-UDP send`.

##### TCP Dial and Send

On each write to a peer, `kadcast.Writer` establishes a new TCP connection, sends the `Kadcast Wire Message` and closes the connection. As this is pretty heavy approach, it's useful on testbed testing where latency is ideal.

##### Raptor Code UDP

RC-UDP (Raptor Code UDP) is UDP-based protocol where each UDP packet on the wire packs a single `encoding symbol`.
See also  `pkg/util/nativeutils/rcudp/README.md`
 
## Kadcast Wire Messages
--------------

Kadcast wire message contains of a header and a payload. While header is mandatory and it has the same structure for all types of messages, payload depends on msgType and could be empty.

**Message Header structure**
|  	|  	|  |  	|	|	|
|-	|-	|-	|-	|-	|-	|
|  Size | 1	|  16|  4	| 2	| 2
| Desc | MsgType | SrcPeerID |SrcPeerNonce | SrcPeerPort | Reserved

 
 MsgType:

|  	|  	|
|-	|-	|
| Ping | 0x00 |
| Pong | 0x01 |
| FindNodes | 0x02 |
| Nodes | 0x03 |
| Broadcast | 0x0A |


**Broadcast Message Payload**

|  	|  	|  	|
|-	| -	| -	|
|  Size	|  1 	|  up to 250000	|
|  Desc	| Kadcast Height | DUSK_PROTOCOL_FRAME  

**Nodes Message Payload**

|  	|  	|  	|  	|  	|  	|
|-	| -	| -	|-	| -	| -	|
|  Size	|  2 	|  4	| 2 | 16 | ...
|  Desc	| Entries Number | IP | Port | PeerID| ... 

**Ping Message Payload** \
Empty

**Pong Message Payload** \
Empty

**FindNodes Message Payload** \
Empty

\* The structure of DUSK_PROTOCOL_FRAME is the same as Gossip processor can read. See also `pkg/p2p/wire/protocol`


## Maintainer and Kademlia Routing State
--------------------

Maintainer is a component that is responsible to build and maintain the Kadcast Routing State. It is a UDP server that handles the folling messages Types - Ping, Pong, FindNodes and FindNodes.

### FindNodes-Nodes Message Flow  (pseudo peers A and B)

1. Peer_A sends `FindNodes message` with its PeerID
2. Peer_B `Maintainer` handles `FindNodes`
3. Peer_B `Maintainer` registers Peer_A
4. Peer_B `Maintainer` tries to get `K` closest peers to `Peer_A`
5. Peer_B `Maintainer` responds with `Nodes` message with K_Closest_Peers list and its own PeerID
6. Peer_A handles `Nodes` message
7. Peer_A registers Peer_B
7. Peer_A starts `Ping-Pong` message flow for each received PeerID from K_Closest_Peers list 

### Ping-Pong Message Flow (pseudo peers A and B)

1. Peer_A sends `Ping message` with its PeerID
2. Peer_B `Maintainer` handles Ping message from PeerA.
3. Peer_B `Maintainer` registers `PeerA`
4. Peer_B `Maintainer` responses with `Pong` message that includes its PeerID
5. Peer_A `Maintainer` handles `Pong` message and registers Peer_B

## Collecting Message
--------------


1. On receiving Kadcast Wire Message, kadcast.Reader (TCPReader or RaptorCodeReader)
2. `kadcast.Writer` handles `topics.Kadcast` event
3. `kadcast.Writer` serialize the event into `Kadcast Wire Message` of type `Broadcast`
4. kadcast.Writer performs kadcast propagation algorithm where transport protocol is
 - ` RC-UDP`, if `kadcast.raptor=true` (config)
- `TCP Dial and Send`, if `kadcast.raptor=false`
