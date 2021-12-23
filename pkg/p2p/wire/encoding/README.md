# [pkg/p2p/wire/encoding](./pkg/p2p/wire/encoding)

The encoding library provides methods to serialize different data types for
storage and transfer over the wire protocol. The library wraps around the
standard `Read` and `Write` functions of an `io.Reader` or `io.Writer`, to
provide simple methods for any basic data type. The library is built to mimic
the way the `binary.Read` and `binary.Write` functions work, but aims to
simplify the code to gain processing speed, and avoid reflection-based
encoding/decoding. As a general rule, for encoding a variable, you should pass
it by value, and for decoding a value, you should pass it by reference. For an
example of this, please refer to the tests or any implementation of this library
in the codebase.

<!-- ToC start -->

## Contents

<!-- ToC end -->

## Methods

### Integers \(integers.go\)

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

### CompactSize integers \(varint.go\)

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

### Booleans \(miscdata.go\)

Booleans can be written and read much like integers, but they have their own
function so that they can be passed without any conversion beforehand. The
functions are:

* `ReadBool(r io.Reader, b *bool) error` to read a bool
* `WriteBool(w io.Writer, b bool) error` to write a bool

Boolean functions use the standard template as demonstrated at the Integers
section.

### 256-bit and 512-bit data structures \(miscdata.go\)

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

### Strings and byte slices \(vardata.go\)

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

### Arrays/slices

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

### Structs

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

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
