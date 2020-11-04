## Score Generator

### Abstract

For more information on the Blind Bid, refer to the [generation package readme](../readme.md).

### Values

#### Score Event

| Field    | Type                   |
| -------- | ---------------------- |
| Proof    | []byte (variable size) |
| Score    | []byte (32 bytes)      |
| Z        | []byte (32 bytes)      |
| Bid List | []byte (variable size) |
| Seed     | BLS Signature          |

### Architecture

The score generator simply listens for `Generation` messages, which triggers proof generation. Upon completion, the resulting proof and it's peripheral information is packaged in a `ScoreEvent`, and published internally.

#### Proof generation

The zk-proof generation is delegated to a dedicated process written in _rustlang_ which communicates with the main _golang_ process through _named pipes_ (also known as `FIFO`). Named pipes have been chosen over other `IPC` methodologies due to the excellent performance they offer. According to our benchmarks, _named pipes_ is two times more performant than _CGO_ (which allows _golang_ to perform calls to _C_ libraries) and comparable to calling the _rust_ process directly.
