## Message encoding

Messages sent over the wire are **COBS encoded** frames of data, with the following structure:

| Field | Size (bytes) |
| --- | --- |
| Magic | 4 |
| Topic | 15 |
| Payload | Any |

These packets are delimited with a 0 byte.

## Topics

Below is a list of supported topics which can be sent and received over the wire:
- Version
- VerAck
- Inv
- GetData
- GetBlocks
- Block
- Tx
- Candidate
- Score
- Reduction
- Agreement
