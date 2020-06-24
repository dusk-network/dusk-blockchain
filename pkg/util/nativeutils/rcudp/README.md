# Raptor code over User Datagram Protocol

RC-UDP provides two utilities a reader and a writer.


## Writer

rcudp.Writer() function is a udp client that uses `gofountain` raptor code encoder to break down a message into a set of blocks. A block together with metadata is packed as a wire packet with following structure:


```
    	MessageID           8 bytes
	NumSourceSymbols    2 bytes
	PaddingSize         2 bytes
	TransferLength      4 bytes
	BlockID             4 bytes
	BlockData        1452 bytes
```

Note that overall size of a packet is up to (=1500-8-20) bytes. (default MTU size minus UDP Header size minus IPv4 header size)

### Writer configs

backoffTimeout (config.go) - defines delay before each UDP socket write. This is intended to reduce the load on both sender and recv udp buffers. See also `net.core.rmem_max`,  `net.core.wmem_max`.

## Reader

UDPReader struct is a UDP server that collects raptor code blocks from the wire and makes attempts to decode (recover) the complete and original message. It correlates blocks by messageID. When a message is decoded, its processing is delegated to the `MessageCollector` (a callback).

An exceptional case is when a message has not been decoded within StaleTimeout time window. In that situation, message is marked as stale and deleted without being ever decoded.


#### Naming convention

In this package `a message` means the data blob to be transmitted on the wire. `Packet` is the UDP packet which includes `a block`. Blocks are the segments which `a message` is encoded to.

