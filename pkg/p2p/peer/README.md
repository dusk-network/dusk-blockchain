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

## Common structures

This section defines the common structures that are used in wire messages. Field types are denoted as their golang types.

### VarInt

Much like VarInt in Bitcoin, it is used to encode variable sized numbers to save space.

| Value | Size | Format |
| --- | --- | --- |
| < 0xFD | 1 | uint8 |
| <= 0xFFFF | 3 | 0xFD followed by a uint16 |
| <= 0xFFFFFFFF | 5 | 0xFE followed by a uint32 |
| > 0xFFFFFFFF | 9 | 0xFF followed by a uint64 |

### protocol.Version

| Field Size | Title | Data Type | 
| --- | --- | --- |
| 1 | Major | uint8 |
| 1 | Minor | uint8 |
| 2 | Patch | uint16 |

Protocol version, expressed in a semver format.

### Block Certificate

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 33 | Batched Signature | []byte | Batched BLS Signature of all votes |
| 8 | Committee Bitset | uint64 | The committee members who voted, represented with 0 or 1 |
| 1 | Step | uint8 | The step at which this committee was formed |

A block certificate, proving that this block was fairly decided upon during consensus. Equivalent to a proof-of-work in Bitcoin.

### peermsg.InvVect

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 1 | Type | uint8 | Inventory type identifier |
| 32 | Hash | []byte | Inventory item hash |

### transactions.Input

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 32 | Key Image | []byte | Image of the key used to sign the transaction |
| 32 | TxID | []byte | TxID of the transaction this input is coming from |
| 1 | Index | uint8 | The position at which this input is placed at in the current transaction |
| ?? | Signature | []byte | Ring signature of the transaction |

### transactions.Output

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| ?? | Commitment | []byte | Pedersen commitment of the amount being transacted |
| 32 | Destination Key | []byte | The one-time public key of the address that the funds are going to |
| ?? | Range proof | []byte | Bulletproof range proof proving that the amount is between 0 and 2^64 |

## Message outline

Below follows an outline and description for each message that can be sent and received over the wire. Field types are denoted as their golang types.

### Version

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 4 | Version | protocol.Version | The version of the Dusk protocol that this node is running. Formatted as semver | 
| 8 | Timestamp | int64 | UNIX timestamp of when the message was created |
| 4 | Service flag | uint32 | Identifier for the services this node offers |

A version message, which is sent when a node attempts to connect with another node in the network. The receiving node sends it's own version message back in response. Nodes should not send any other messages to each other until both of them have sent a version message.

### VerAck

This message is sent as a reply to the version message, to acknowledge a peer has received and accepted this version message. It contains no other information.

### Inv

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 1-9 | Count | VarInt | Amount of items contains in the inventory message |
| 33 * Count | Inv List | []peermsg.InvVect | Inventory items | 

Inventory messages are used to advertise transactions and blocks to the network. It can be received unsolicited, or as a reply to GetBlocks.

### GetData

A GetData message is sent as a response to an inventory message, and should contain the hashes of the items that the peer wishes to receive the data for. It is structed exactly the same as the Inv message, only the header topic differs.

### GetBlocks

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 1-9 | Count | VarInt | Amount of locators |
| 32 * Count | Locators | [][]byte | Locator hashes, revealing a node's last known block | 

A GetBlocks message is sent when a block is received which has a height that is further than 1 apart from the currently known highest block. When a GetBlocks is sent, an Inv is returned containing up to 500 block hashes that the requesting peer is missing, which it can then download with GetData.

### Block

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 1 | Version | uint8 | Block version byte |
| 8 | Height | uint64 | |
| 8 | Timestamp | int64 | |
| 32 | Previous Block Hash | []byte | |
| 33 | Seed | []byte | BLS Signature of the previous block seed, made by a block generator |
| 32 | Tx Root | []byte | Merkle root hash of all transactions in this block |
| 42 | Certificate | Block Certificate | |
| 32 | Hash | []byte | Hash of this block |
| ?? | Txs | []tx | All transactions contained in this block |

A Dusk block. Note that the certificate is generated during consensus. For this reason, this field is empty when a block is transmitted with a Candidate message.

The block hash is the hash of the following fields:
- Version
- Height
- Timestamp
- Previous block hash
- Seed

### Tx

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 1 | Type | uint8 | Transaction type identifier |
| 1 | Version | uint8 | Transaction version byte |
| 32 | TxID | []byte | Transaction identifier (hash of transaction fields) |
| ?? | Inputs | []transactions.Input | |
| ?? | Outputs | []transactions.Output | |
| 8 | Fee | uint64 | Amount of DUSK paid in fees |
| ?? | Optional Fields | ?? | |

A Dusk transaction. Note that this only defines the standard transaction, as most other types contain extra information on top of the defined fields.

The TxID is calculated by hashing all of the defined fields. The optional fields are not included in the TxID hash.

### Candidate

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| ?? | Block | Block | |

A block proposal, made by a block generator. Note that the certificate field is empty when a block is included in a Candidate message, as consensus has not yet been reached on it. Should come directly after a Score message.

### Score

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 8 | Round | uint64 | Consensus round |
| 32 | Score | []byte | Proof score |
| ?? | Proof | []byte | Zero-knowledge proof of blind bid |
| 32 | Z | []byte | Identity hash | 
| 32 * ?? | Bidlist Subset | []byte | Subset of bids, that the proof generator's bid is shuffled into |
| 33 | Seed | []byte | Seed of the proposed block |
| 32 | Hash | []byte | Hash of the proposed block |

A consensus score message, generated by a block generator. This score message determines whether or not the proposed block is chosen by the provisioners. Should precede a Candidate message. 

### Reduction

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 129 | BLS Public Key | []byte | Sender's BLS Public Key |
| 8 | Round | uint64 | Consensus round |
| 1 | Step | uint8 | Consensus step |
| 32 | Block Hash | []byte | The block hash that this node is voting on |
| 33 | Signed Hash | []byte | Compressed BLS Signature of the block hash field | 

A reduction message, sent by provisioners during consensus. It is essentially a vote for one of the Candidate blocks they received prior to the reduction phase.

### Agreement

| Field Size | Title | Data Type | Description |
| --- | --- | --- | --- |
| 129 | BLS Public Key | []byte | Sender's BLS Public Key |
| 8 | Round | uint64 | Consensus round |
| 1 | Step | uint8 | Consensus step |
| 32 | Block Hash | []byte | The block hash that these nodes have voted on |
| 33 | Signed Votes | []byte | BLS Signature of the Votes field |
| 2*?? | Votes | []StepVotes | The compressed representation of the votes that were cast during a step |

An agreement message, sent by provisioners during consensus. It's a compressed collection of all the votes that were cast in 2 steps of reduction, and is paramount in reaching consensus on a certain block.
