# [pkg/util/nativeutils/rcudp](./pkg/util/nativeutils/rcudp)

Simple short-message (max one UDP packet) reliable transport protocol using
Raptor FEC codes for retransmit-avoidance.

<!-- ToC start -->

## Contents

1. [Raptor code over User Datagram Protocol](#raptor-code-over-user-datagram-protocol)
    1. [Writer](#writer)
    1. [## Writer](#-writer)
        1. [Tuning](#tuning)
    1. [Reader](#reader)
    1. [## Reader](#-reader)
        1. [Tuning](#tuning-1)

<!-- ToC end -->

# Raptor code over User Datagram Protocol

RC-UDP implements Raptor Coding over UDP transport protocol with tunable
overhead.

The package provides two utilities - a RC-UDP Reader (Client) and a RC-UDP
Writer (Server). Message encoding/decoding is based on the Raptor fountain
code (also called the R10 code) from RFC 5053.

Naming convention

In this package `a message` means the complete data blob to be transmitted on
the wire. \
`Packet` is the UDP packet which includes `a single block` of the message. \
Blocks are the segments which `a message` is encoded to.

## Writer
----------------

`rcudp.Writer(...)` function is a UDP client that uses `gofountain` Raptor Code
encoder to split a message into source blocks. An encoded block together with
metadata is marshalled into a UDP packet with following structure:

```
    MessageID           8 bytes
	NumSourceSymbols    2 bytes
	PaddingSize         2 bytes
	TransferLength      4 bytes
	BlockID             4 bytes
	BlockData        1452 bytes
```

Note that overall size of a packet is up to 1472 (=1500-8-20)
bytes. (`default MTU size` minus `UDP Header size` minus `IPv4 header size`)

The size of a complete wire message is `TransferLength` - `PaddingSize`. A
limitation is that the code supports a maximum of 8192 source blocks. With
current BlockData size of 1452, the maximum length of a message that can be
transmitted is 1452*8192 (~11.89 MB)

`NumSourceSymbols` - K. Must be in the range [4, 8192] (inclusive). This is how
many source symbols the input message will be divided into.

### Tuning

`backoffTimeout` (config.go) - defines delay before each UDP socket write. This
is intended to reduce the load on both sender and recv udp buffers.

`writeBufferSize` (config.go) - UDP Sender buffer size can be up
to `net.core.wmem_max` (Linux)

`redundancyFactor` input param - defines the count of additional encoded blocks
to be generated and sent

## Reader
-----

`UDPReader` is a UDP server that collects Raptor fountain code blocks from the
wire and makes attempts to decode (recover) the complete and original message.
It correlates blocks by `MessageID`. When a message is decoded, its processing
is delegated to the `MessageCollector` (a callback).

### Tuning

`staleTimeout` - The size of the time-window within which a message should be
completely received and decoded. Out of this time-window, the message is marked
as stale and deleted.

`readBufferSize` - UDP Recv buffer size can be up to `net.core.rmem_max` (Linux)

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
