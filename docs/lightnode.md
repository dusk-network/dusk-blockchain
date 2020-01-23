## Light-node mode

The node offers an option for the user to start up the node in what's called 'light-node' mode. It is intended for those wishing to only use the wallet functionality, and disables certain components, and allows the node to ignore most of the network messages, drastically decreasing bandwidth consumption. Below follows a more in-depth explanation of the effects 'light-node' mode has on the process.

### Disabled components

When in 'light-node' mode, a set of components are not initialized on startup:

- The consensus module
- The candidate component
- The mempool

Furthermore, it restricts wallet functionality. Any commands that have to do with consensus will return an error message when they are called. These commands are as follows:

- `bid`
- `stake`
- `unconfirmedbalance`
- `automateconsensustxs`
- `viewmempool`

### Network setup

The network setup will differ from that of a 'full node'. To start off, in the `version` message which is exchanged during handshake, the `ServiceFlag` field will now reflect which mode the node is in (full/light). This lets the receiving node know what messages to expect, and what messages to forward to this node. When in light-node mode, a node can only receive the following messages:

- `Inv` (strictly containing block data, tx data is ignored)
- `Block`

Similarly, it can only send the following messages out to others:

- `Tx`
- `GetData` (also strictly for block data)

Any other messages received from a light node are consequently ignored.

On an implementation level, the restriction of sending messages happens through the `GossipConnector`. When a message is pulled from the ring buffer, the `GossipConnector` first checks the peer's service flag, and from there, decides whether or not to forward the message.

As for receiving messages, the `peer.Router` implementation differs between the different modes. A full node will use the standard `peer.messageRouter`, though there are extra guards in place which disallow the node to respond to messages from light nodes which do not match the topic list mentioned above. The light node utilizes a more light-weight `peer.lightRouter` which only concerns itself with the topics mentioned above, discarding any other message.

## Future work

In a brief discussion that was had about the implementation, Matteo suggested to take a different, more extensible approach in the future, as having what's known as a 'branching codebase' is considered to be a code smell. The idea is simply, instead of including the service flag in the handshake message, including a bitmask of message topics which can be sent and received by the node in question. Going off of this bitmask, a receiving node can then accept messages which match it, or drop messages which don't. Similarly, a sending node can decide whether or not to send a certain message, by checking it against this bitmask. This would allow for greater extensibility of perhaps different node types in the future (the block explorer comes to mind), and rids us of the single config flag, which branches the codebase.
