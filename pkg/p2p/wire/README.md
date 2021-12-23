# [pkg/p2p/wire](./pkg/p2p/wire)

Being otherwise empty, this *folder also contains encoder specification document
and the Event and Store interfaces.*

<!-- ToC start -->
##  Contents

   1. [Encoding](#encoding)
      1. [Methods](#methods)
         1. [Integers \(integers.go\)](#integers-\integersgo\)
         1. [CompactSize integers \(varint.go\)](#compactsize-integers-\varintgo\)
         1. [Booleans \(miscdata.go\)](#booleans-\miscdatago\)
         1. [256-bit and 512-bit data structures \(miscdata.go\)](#256-bit-and-512-bit-data-structures-\miscdatago\)
         1. [Strings and byte slices \(vardata.go\)](#strings-and-byte-slices-\vardatago\)
         1. [Arrays/slices](#arrays/slices)
         1. [Structs](#structs)
   1. [Message encoding](#message-encoding)
   1. [Topics](#topics)
   1. [Common structures](#common-structures)
      1. [VarInt](#varint)
      1. [protocol.Version](#protocolversion)
      1. [Block Certificate](#block-certificate)
      1. [message.InvVect](#messageinvvect)
      1. [transactions.Input](#transactionsinput)
      1. [transactions.Crossover](#transactionscrossover)
      1. [transactions.Output](#transactionsoutput)
      1. [transactions.Fee](#transactionsfee)
      1. [StepVotes](#stepvotes)
   1. [Message outline](#message-outline)
      1. [Version](#version)
      1. [VerAck](#verack)
      1. [Ping](#ping)
      1. [Pong](#pong)
      1. [Inv](#inv)
      1. [GetData](#getdata)
      1. [GetBlocks](#getblocks)
      1. [Block](#block)
      1. [Tx](#tx)
      1. [MemPool](#mempool)
      1. [Candidate](#candidate)
      1. [GetCandidate](#getcandidate)
      1. [NewBlock](#newblock)
      1. [Reduction](#reduction)
      1. [Agreement](#agreement)
<!-- ToC end -->

## Encoding

The encoding library provides methods to serialize different data types for
storage and transfer over the wire protocol. The library wraps around the
standard `Read` and `Write` functions of an `io.Reader` or `io.Writer`, to
provide simple methods for any basic data type. The library is built to mimick
the way the `binary.Read` and `binary.Write` functions work, but aims to
simplify the code to gain processing speed, and avoid reflection-based
encoding/decoding. As a general rule, for encoding a variable, you should pass
it by value, and for decoding a value, you should pass it by reference. For an
example of this, please refer to the tests or any implementation of this library
in the codebase.

### Methods

#### Integers \(integers.go\)

Functions are provided for integers of any size \(between 8 and 64 bit\). All
integers will need to be passed and retrieved as their unsigned format. Signed
integers can be converted to and from their unsigned format by simple type
conversion.

The functions are declared like this:

**Reading**

* `ReadUint8(r io.Reader, v *uint8) error`
* `ReadUint16(r io.Reader, o binary.ByteOrder, v *uint16) error`
* `ReadUint32(r io.Reader, o binary.ByteOrder, v *uint32) error`
* `ReadUint64(r io.Reader, o binary.ByteOrder, v *uint64) error`

**Writing**

* `WriteUint8(w io.Writer, v uint8) error`
* `WriteUint16(w io.Writer, o binary.ByteOrder, v uint16) error`
* `WriteUint32(w io.Writer, o binary.ByteOrder, v uint32) error`
* `WriteUint64(w io.Writer, o binary.ByteOrder, v uint64) error`

Usage of this module will be as follows. Let's take a uint64 for example:

```go
// Error handling omitted for clarity
var x uint64 = 5

buf := new(bytes.Buffer)

err := encoding.PutUint64(buf, binary.LittleEndian, x)

// buf.Bytes() = [5 0 0 0 0 0 0 0]

var y uint64
err := encoding.Uint64(buf, binary.LittleEndian, &y)

// y = 5
```

For an int64 you would write:

```go
// Error handling omitted for clarity
var x int64 = -5

buf := new(bytes.Buffer)

err := encoding.PutUint64(buf, binary.LittleEndian, uint64(x))

// buf.Bytes() = [251 255 255 255 255 255 255 255]

var y uint64
err := encoding.Uint64(buf, binary.LittleEndian, &y)
z := int64(y)

// z = -5
```

The encoding module uses this same function template for all other data
structures, _except for CompactSize integers_, which are explained below.

#### CompactSize integers \(varint.go\)

The library provides an implementation of CompactSize integers as described in
the [Bitcoin Developer Reference](https://bitcoin.org/en/developer-reference#compactsize-unsigned-integers)
. The functions implemented are as follows:

* `ReadVarInt(r io.Reader) (uint64, error)` for reading CompactSize integers
* `WriteVarInt(w io.Writer, v uint64) error` for writing CompactSize integers
* `VarIntEncodeSize(v uint64) error` for determining how many bytes an integer
  will take up through CompactSize encoding

For reading and writing CompactSize integers, the functions make use of the
aforementioned integer serialization functions. CompactSize integers are best
used when serializing an array of variable size, for example when serializing
the inputs and outputs of a transaction.

A quick usage example:

```go
// Error handling omitted for clarity

x := []int{5, 2, 98, 200}
l := len(x)
buf := new(bytes.Buffer)
err := encoding.WriteVarInt(buf, uint64(l))

// And then, to get it out...
n, err := encoding.ReadVarInt(buf)
```

#### Booleans \(miscdata.go\)

Booleans can be written and read much like integers, but they have their own
function so that they can be passed without any conversion beforehand. The
functions are:

* `ReadBool(r io.Reader, b *bool) error` to read a bool
* `WriteBool(w io.Writer, b bool) error` to write a bool

Boolean functions use the standard template as demonstrated at the Integers
section.

#### 256-bit and 512-bit data structures \(miscdata.go\)

For data that is either 256 or 512 bits long \(such as hashes and signatures\),
separate functions are defined.

For 256-bit data:

* `Read256(r io.Reader, b *[]byte) error` to read 256 bits of data
* `Write256(w io.Writer, b []byte) error` to write 256 bits of data

For 512-bit data:

* `Read512(r io.Reader, b *[]byte) error` to read 512 bits of data
* `Write512(w io.Writer, b []byte) error` to write 512 bits of data

256-bit and 512-bit functions use the standard template as demonstrated at the
Integers section.

#### Strings and byte slices \(vardata.go\)

Functions in this source file will make use of CompactSize integers for
embedding and reading length prefixes. For byte slices of variable length there
are two functions.

* `ReadVarBytes(r io.Reader, b *[]byte) error` to read byte data
* `WriteVarBytes(w io.Writer, b []byte) error` to write byte data

For strings of variable length \(like a message\) there are two convenience
functions, which point to the functions above.

* `ReadString(r io.Reader, s *string) error`
* `WriteString(w io.Writer, s string) error`

Writing a string will simply convert it to a byte slice, and reading it will
take the byte slice and convert it to a string.

String and byte slice functions use the standard template as demonstrated at the
Integers section.

#### Arrays/slices

For arrays or slices, you should use a CompactSize integer to denote the length
of the array, followed by the data, serialized in a for loop.

The method is as follows \(I will take a slice of strings for this example\):

```go
// Error handling omitted for clarity

slice := []string{"foo", "bar", "baz"}

buf := new(bytes.Buffer)
err := encoding.WriteVarInt(buf, uint64(len(slice)))
// Handle error

for _, str := range slice {
err := encoding.WriteString(buf, str)
}

// buf.Bytes() = [3 3 102 111 111 3 98 97 114 3 98 97 122]

count, err := ReadVarInt(buf)
// Handle error

// Make new array with len 0 and cap count
newSlice := make([]string, 0, count)
for i := uint64(0); i < count; i++ {
err := encoding.ReadString(buf, *newSlice[i])
}

// newArr = ["foo" "bar" "baz"]
```

#### Structs

For structs, the best way of doing this is to create an `Encode()`
and `Decode()` method. In these functions you should go off each field one by
one, and encode them into the buffer, or decode them one by one from the buffer,
and then populate the respective fields.

A more advanced example can be found in the transactions folder. I will
demonstrate a small one here:

```go
// Error handling omitted for clarity

type S struct {
Str string
A []uint64
B bool
}

func (s *S) Encode(w io.Writer) error {
// Str
err := encoding.WriteString(w, s.Str)

// A
err := encoding.WriteVarInt(w, uint64(len(s.A)))
for _, n := range s.A {
err := encoding.WriteUint64(w, binary.LittleEndian, n)
}

// B
err := encoding.WriteBool(w, s.B)
}

func (s *S) Decode(r io.Reader) error {
// Str
err := encoding.ReadString(r, &s.Str)

// A
count, err := encoding.ReadVarInt(r)

s.A = make([]uint64, 0, count)
for i := uint64(0); i < length; i++ {
n, err := encoding.ReadUint64(r, binary.LittleEndian, &s.A[i])
}

// B
err := encoding.ReadBool(r, &s.B)

return nil
}
```

As a sidenote for structs, it's advisable to include a function
like `GetEncodeSize()` as seen in the stealthtx source file. Knowing how long
the buffer has to be before running `Encode()` will reduce allocations during
encoding, and increase performance \(especially with transactions and blocks as
they will contain many different fields\).

```go
count := StealthTX.GetEncodeSize()
bs := make([]byte, 0, count)
buf := bytes.NewBuffer(bs)
err := StealthTX.Encode(buf)
```



todo: there might be some duplication as this page merges two nearly similar
documents that didn't live in the same place.

## Message encoding

Messages sent over the wire are length-prefixed frames of data, with the
following structure:

| Field | Size \(bytes\) |
| :--- | :--- |
| Packet Length | 8 |
| Magic | 4 |
| Reserved | 8 |
| Checksum | 4 |
| Topic | 1 |
| Payload | Any |

## Topics

Below is a list of supported topics which can be sent and received over the
wire:

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
* NewBlock
* Reduction
* Agreement

## Common structures

This section defines the common structures that are used in wire messages. Field
types are denoted as their golang types.

### VarInt

Much like VarInt in Bitcoin, it is used to encode variable sized numbers to save
space.

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

A block certificate, proving that this block was fairly decided upon during
consensus. Equivalent to a proof-of-work in Bitcoin.

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

Below follows an outline and description for each message that can be sent and
received over the wire. Field types are denoted as their golang types.

### Version

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 4 | Version | protocol.Version | The version of the Dusk protocol that this node is running. Formatted as semver |
| 8 | Timestamp | int64 | UNIX timestamp of when the message was created |
| 4 | Service flag | uint32 | Identifier for the services this node offers |

A version message, which is sent when a node attempts to connect with another
node in the network. The receiving node sends it's own version message back in
response. Nodes should not send any other messages to each other until both of
them have sent a version message.

### VerAck

This message is sent as a reply to the version message, to acknowledge a peer
has received and accepted this version message. It contains no other
information.

### Ping

This message is used to inquire if a node is still active and maintaining its
connection on the other end. The Ping message contains no other information.

### Pong

This message is used to respond to other nodes' Ping messages, informing them
that a connection is still being maintained. It carries no other information.

### Inv

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 1-9 | Count | VarInt | Amount of items contains in the inventory message |
| 33 \* Count | Inv List | \[\]peermsg.InvVect | Inventory items |

Inventory messages are used to advertise transactions and blocks to the network.
It can be received unsolicited, or as a reply to GetBlocks.

### GetData

A GetData message is sent as a response to an inventory message, and should
contain the hashes of the items that the peer wishes to receive the data for. It
is structed exactly the same as the Inv message, only the header topic differs.

### GetBlocks

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 1-9 | Count | VarInt | Amount of locators |
| 32 \* Count | Locators | \[\]\[\]byte | Locator hashes, revealing a node's last known block |

A GetBlocks message is sent when a block is received which has a height that is
further than 1 apart from the currently known highest block. When a GetBlocks is
sent, an Inv is returned containing up to 500 block hashes that the requesting
peer is missing, which it can then download with GetData.

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

A Dusk block. Note that the certificate is generated during consensus. For this
reason, this field is empty when a block is transmitted with a Candidate
message.

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

This message is used to request the mempool contents of another node in the
network. Used by nodes which just recently joined, and wish to have an updated
view of the current pending transaction queue. Responses are typically given
with `Inv` messages. The message carries no other information.

### Candidate

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| ?? | Block | Block |  |

A standalone candidate block provided to a requesting node. This message will
only be sent out in response to a `GetCandidate`, as the candidate block is
normally included in a `NewBlock` message.

### GetCandidate

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 32 | Block Hash | \[\]byte | Hash of the requested candidate block |

This message can be used by nodes which are missing a certain candidate block at
any point during consensus. Responses will be of the `Candidate` topci.

### NewBlock

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 129 | Public Key | \[\] | BLS Public Key of the sender |
| 8 | Round | uint64 | Consensus round |
| 1 | Step | uint8 | Consensus step |
| 32 | Block Hash | \[\]byte | Hash of the candidate block |
| 32 | Identity | \[\]byte | Identity hash |
| 33 | Seed | \[\]byte | Seed of the proposed block |
| 32 | Previous Hash | \[\]byte | Hash of the previous block |
| ?? | Candidate | Block | Candidate block |

A consensus NewBlock message, generated by an extracted Provisioner of the
Selection step. It propagates the Candidate block.

### Reduction

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 129 | BLS Public Key | \[\]byte | Sender's BLS Public Key |
| 8 | Round | uint64 | Consensus round |
| 1 | Step | uint8 | Consensus step |
| 32 | Block Hash | \[\]byte | The block hash that this node is voting on |
| 33 | Signed Hash | \[\]byte | Compressed BLS Signature of the block hash field |

A reduction message, sent by provisioners during consensus. It is essentially a
vote for one of the Candidate blocks they received prior to the reduction phase.

### Agreement

| Field Size | Title | Data Type | Description |
| :--- | :--- | :--- | :--- |
| 129 | BLS Public Key | \[\]byte | Sender's BLS Public Key |
| 8 | Round | uint64 | Consensus round |
| 1 | Step | uint8 | Consensus step |
| 32 | Block Hash | \[\]byte | The block hash that these nodes have voted on |
| 2\*?? | Votes | \[\]StepVotes | The compressed representation of the votes that were cast during a step |
| ?? | Committee Representation | big.Int | Bitmap representation of committee members which voted |

An agreement message, sent by provisioners during consensus. It's a compressed
collection of all the votes that were cast in 2 steps of reduction, and is
paramount in reaching consensus on a certain block.


Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
