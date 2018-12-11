# Dusk wire protocol

Found in the `wire` folder, the Dusk wire protocol is based on the [Bitcoin wire protocol](https://en.bitcoin.it/wiki/Protocol_documentation), with the source code modeled after [Kev's NEO wire protocol implementation](https://github.com/decentralisedkev/neo-go/tree/v2/pkg/wire) and [the btcd wire protocol implementation](https://github.com/btcsuite/btcd/tree/master/wire). The `payload` package defines different types of messages, and provides encoding/decoding methods for all of them, while the `wire` package offers generalised `WriteMessage` and `ReadMessage` functions, and a function to get the machine's external IP (`GetLocalIP`). Tests are included for all completed source files, to ensure proper encoding and decoding of information and commands.

## Usage

### Writing a message

To prepare a message for writing over the wire, you can use the New<message type> function. For example, if we were preparing to send a `mempool` message (which has no further payload), it would look like this:

```go
msg := payload.NewMsgMemPool()
```

For messages that do carry a payload, it will be either taken in through the creation function, or in some cases through extra functions. Let's take a `version` message for example:

```go
// Let's say we have established a TCP connection with a peer here

// Make NetAddress structs for `from` and `to` fields
from := payload.NewNetAddress(wire.GetLocalIP(), cfg.Port)
to := payload.NewNetAddress(peer.Addr, peer.Port)

// This will create a `version` message with the current running protocol version, as well as the local and remote network addresses formatted as NetAddress structs
msg := payload.NewMsgVersion(wire.ProtocolVersion, from, to)
```

Lastly, in the case of a message having methods to add information to it, you can simply call it on the message itself. Let's take an `addr` message as an example:

```go
msg := payload.NewMsgAddr()

// Now let's say we wanted to add the `from` and `to` addresses from the last code block
msg.AddAddr(from)
msg.AddAddr(to)
```

**Check out the actual code for full specifications for every message**

Once your message is ready to be sent, you can do so with the `WriteMessage` function.

```go
// Error handling omitted for clarity
wire.WriteMessage(peer.Conn, wire.DevNet, msg)
```

This function will then prepend a header to your message, encode your message to binary and send it over the wire, to the specified peer.

### Reading a message

A received message can be read from the wire with the `ReadMessage` function.

```go
// Error handling omitted for clarity
payload, err := wire.ReadMessage(peer.Conn, wire.DevNet)
```

This will decode the message header and then use the header information to properly decode the payload. It then returns the payload, which is denoted as an interface value. We can identify the payload type by doing a type assertion like so:

```go
switch msg := payload.(type) {
    case *payload.MsgVersion:
        // Handle
    case *payload.MsgVerAck:
        // Handle
    // etc...
}
```

From here on you should be able to handle the message accordingly.
