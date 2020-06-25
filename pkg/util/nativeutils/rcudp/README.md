# Raptor code over User Datagram Protocol

RC-UDP implements Raptor Coding over UDP transport protocol with tunable overhead. 


The package provides two utilities - a RC-UDP Reader (Client) and a RC-UDP Writer (Server). Message encoding/decoding is based on the Raptor fountain code (also called the R10 code) from RFC 5053.


Naming convention

In this package `a message` means the complete data blob to be transmitted on the wire. \
 `Packet` is the UDP packet which includes `a single block` of the message. \
 Blocks are the segments which `a message` is encoded to.

## Writer
----------------

`rcudp.Writer(...)` function is a udp client that uses `gofountain` raptor code encoder to break down a message into a set of blocks. A block together with metadata is packed as a UDP packet with following structure:


```
    MessageID           8 bytes
	NumSourceSymbols    2 bytes
	PaddingSize         2 bytes
	TransferLength      4 bytes
	BlockID             4 bytes
	BlockData        1452 bytes
```

Note that overall size of a packet is up to 1472 (=1500-8-20) bytes. (`default MTU size` minus `UDP Header size` minus `IPv4 header size`)

The size of a complete wire message is `TransferLength` - `PaddingSize`.

NumSourceSymbols - `#TODO`

### Tuning

`backoffTimeout` (config.go) - defines delay before each UDP socket write. This is intended to reduce the load on both sender and recv udp buffers.
 
`writeBufferSize` (config.go) - UDP Sender buffer size can be up to `net.core.wmem_max` (Linux)

`redundancyFactor` input param - `#TODO`

## Reader
-----

`UDPReader` is a UDP server that collects Raptor fountain code blocks from the wire and makes attempts to decode (recover) the complete and original message. It correlates blocks by `MessageID`. When a message is decoded, its processing is delegated to the `MessageCollector` (a callback). 


### Tuning


`staleTimeout` - The size of the time-window within which a message should be completely received and decoded. Out of this time-window, the message is marked as stale and deleted.

`readBufferSize` - UDP Recv buffer size can be up to `net.core.rmem_max` (Linux)




