# DUSK Network Wire Protocol Documentation

## Message encoding

Messages sent over the wire are length-prefixed frames of data, with the following structure:

| Field | Size \(bytes\) |
| :--- | :--- |
| Packet Length | 8 |
| Magic | 4 |
| Reserved | 8 |
| Checksum | 4 |
| Topic | 1 |
| Payload | Any |

## Topics

Below is a list of supported topics which can be sent and received over the wire:

* Version
* VerAck
* Ping
* Pong
* Inv
* GetData
* GetBlocks
* Block
* Tx
* MemPool
* Candidate
* GetCandidate
* Score
* Reduction
* Agreement

## Common structures

This section defines the common structures that are used in wire messages. Field types are denoted as their golang types.

### VarInt

Much like VarInt in Bitcoin, it is used to encode variable sized numbers to save space.

| Value | Size | Format |
| :--- | :--- | :--- |
| &lt; 0xFD | 1 | uint8 |
| &lt;= 0xFFFF | 3 | 0xFD followed by a uint16 |
| &lt;= 0xFFFFFFFF | 5 | 0xFE followed by a uint32 |
| &gt; 0xFFFFFFFF | 9 | 0xFF followed by a uint64 |

### protocol.Version

| Field Size | Title | Data Type |
| :--- | :--- | :--- |
| 1 | Major | uint8 |
| 1 | Minor | uint8 |
| 2 | Patch | uint16 |

Protocol version, expressed in a semver format.

### Block Certificate

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 33 | Batched Signature Step 1 | \[\]byte | Batched BLS Signature of all votes of step 1 |
| 33 | Batched Signature Step 2 | \[\]byte | Batched BLS Signature of all votes of step 2 |
| 1 | Step | uint8 | The step at which consensus was reached |
| 8 | Committee Bitset Step 1 | uint64 | The committee members who voted on step 1, represented with 0 or 1 |
| 8 | Committee Bitset Step 2 | uint64 | The committee members who voted on step 2, represented with 0 or 1 |

A block certificate, proving that this block was fairly decided upon during consensus. Equivalent to a proof-of-work in Bitcoin.

### message.InvVect

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 1 | Type | uint8 | Inventory type identifier |
| 32 | Hash | \[\]byte | Inventory item hash |

### transactions.Input

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 32 | Nullifier | \[\]byte | A hash which 'spends' a Note, according to the Phoenix rules |

### transactions.Crossover

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 32 | Value Commitment | \[\]byte | Commitment to the value assigned to the Crossover |
| 32 | Nonce | \[\]byte | Nonce, used for encryption |
| 96 | Encrypted Data | \[\]byte | Encrypted value of the Commitment |

### transactions.Output

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| ?? | Commitment | \[\]byte | Pedersen commitment of the amount being transacted |
| 32 | Destination Key | \[\]byte | The one-time public key of the address that the funds are going to |
| ?? | Range proof | \[\]byte | Bulletproof range proof proving that the amount is between 0 and 2^64 |

### transactions.Fee

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 8 | Gas Limit | uint64 | Amount of gas allowed for VM execution of this transaction |
| 8 | Gas Price | uint64 | Amount of DUSK paid per unit of gas, used for execution of this transaction |
| 32 | R | \[\]byte | Random point of the stealth address |
| 32 | PkR | \[\]byte | Public key of the stealth address |

### StepVotes

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 129 | Aggregated Public Keys | bls.APK | Aggregated representation of all public keys that voted for this step |
| 8 | Bitset | uint64| Bitmap representation of which committee members are included in the APK |
| 33 | Signature | bls.Signature | Aggregated BLS signature |
| 1 | Step | uint8 | The step which was voted on |

## Message outline

Below follows an outline and description for each message that can be sent and received over the wire. Field types are denoted as their golang types.

### Version

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 4 | Version | protocol.Version | The version of the Dusk protocol that this node is running. Formatted as semver |
| 8 | Timestamp | int64 | UNIX timestamp of when the message was created |
| 4 | Service flag | uint32 | Identifier for the services this node offers |

A version message, which is sent when a node attempts to connect with another node in the network. The receiving node sends it's own version message back in response. Nodes should not send any other messages to each other until both of them have sent a version message.

### VerAck

This message is sent as a reply to the version message, to acknowledge a peer has received and accepted this version message. It contains no other information.

### Ping

This message is used to inquire if a node is still active and maintaining its connection on the other end. The Ping message contains no other information.

### Pong

This message is used to respond to other nodes' Ping messages, informing them that a connection is still being maintained. It carries no other information.

### Inv

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 1-9 | Count | VarInt | Amount of items contains in the inventory message |
| 33 \* Count | Inv List | \[\]peermsg.InvVect | Inventory items |

Inventory messages are used to advertise transactions and blocks to the network. It can be received unsolicited, or as a reply to GetBlocks.

### GetData

A GetData message is sent as a response to an inventory message, and should contain the hashes of the items that the peer wishes to receive the data for. It is structed exactly the same as the Inv message, only the header topic differs.

### GetBlocks

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 1-9 | Count | VarInt | Amount of locators |
| 32 \* Count | Locators | \[\]\[\]byte | Locator hashes, revealing a node's last known block |

A GetBlocks message is sent when a block is received which has a height that is further than 1 apart from the currently known highest block. When a GetBlocks is sent, an Inv is returned containing up to 500 block hashes that the requesting peer is missing, which it can then download with GetData.

### Block

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 1 | Version | uint8 | Block version byte |
| 8 | Height | uint64 |  |
| 8 | Timestamp | int64 |  |
| 32 | Previous Block Hash | \[\]byte |  |
| 33 | Seed | \[\]byte | BLS Signature of the previous block seed, made by a block generator |
| 32 | Tx Root | \[\]byte | Merkle root hash of all transactions in this block |
| 83 | Certificate | Block Certificate |  |
| 32 | Hash | \[\]byte | Hash of this block |
| ?? | Txs | \[\]tx | All transactions contained in this block |

A Dusk block. Note that the certificate is generated during consensus. For this reason, this field is empty when a block is transmitted with a Candidate message.

The block hash is the hash of the following fields:

* Version
* Height
* Timestamp
* Previous block hash
* Seed

### Tx

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 1 | Version | uint8 | Transaction version byte |
| 1 | Type | uint8 | Transaction type identifier |
| 32 | Anchor | \[\]byte | State root at the time of transaction creation |
| ?? | Inputs | \[\]transactions.Input | Hashes of notes being spent |
| 160 | Crossover | transactions.Crossover |  |
| ?? | Notes | \[\]transactions.Output | Notes created by this transaction |
| 80 | Fee | uint64 | Amount of DUSK paid in fees |
| ?? | Spending Proof | \[\]byte | Proof of ownership of spent notes in the transaction |
| ?? | Call Data | \[\]byte | Collection of VM instructions and arguments |

### MemPool

This message is used to request the mempool contents of another node in the network. Used by nodes which just recently joined, and wish to have an updated view of the current pending transaction queue. Responses are typically given with `Inv` messages. The message carries no other information.

### Candidate

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| ?? | Block | Block |  |

A standalone candidate block provided to a requesting node. This message will only be sent out in response to a `GetCandidate`, as the candidate block is normally included in a `Score` message.

### GetCandidate

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 32 | Block Hash | \[\]byte | Hash of the requested candidate block |

This message can be used by nodes which are missing a certain candidate block at any point during consensus. Responses will be of the `Candidate` topci.

### Score

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 129 | Public Key | \[\] | BLS Public Key of the sender |
| 8 | Round | uint64 | Consensus round |
| 1 | Step | uint8 | Consensus step |
| 32 | Block Hash | \[\]byte | Hash of the candidate block |
| 32 | Score | \[\]byte | Proof score |
| ?? | Proof | \[\]byte | Zero-knowledge proof of blind bid |
| 32 | Identity | \[\]byte | Identity hash |
| 33 | Seed | \[\]byte | Seed of the proposed block |
| 32 | Previous Hash | \[\]byte | Hash of the previous block |
| ?? | Candidate | Block | Candidate block |

A consensus score message, generated by a block generator. This score message determines whether or not the proposed block is chosen by the provisioners. Should precede a Candidate message.

### Reduction

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 129 | BLS Public Key | \[\]byte | Sender's BLS Public Key |
| 8 | Round | uint64 | Consensus round |
| 1 | Step | uint8 | Consensus step |
| 32 | Block Hash | \[\]byte | The block hash that this node is voting on |
| 33 | Signed Hash | \[\]byte | Compressed BLS Signature of the block hash field |

A reduction message, sent by provisioners during consensus. It is essentially a vote for one of the Candidate blocks they received prior to the reduction phase.

### Agreement

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 129 | BLS Public Key | \[\]byte | Sender's BLS Public Key |
| 8 | Round | uint64 | Consensus round |
| 1 | Step | uint8 | Consensus step |
| 32 | Block Hash | \[\]byte | The block hash that these nodes have voted on |
| 2\*?? | Votes | \[\]StepVotes | The compressed representation of the votes that were cast during a step |
| ?? | Committee Representation | big.Int | Bitmap representation of committee members which voted |

An agreement message, sent by provisioners during consensus. It's a compressed collection of all the votes that were cast in 2 steps of reduction, and is paramount in reaching consensus on a certain block.

