#Encoding

The encoding library provides methods to serialize different data types for storage and transfer over the wire protocol.
The library is mostly built to enhance performance of the operations needed to do this. Though not standard, I believe
this way of setting it up will end up increasing performance quite a bit for the software as it is running - for example
while a node will be synchronizing (deserializing a lot of blocks until it is up to the current height) or just while
it is validating, especially once the network will start to increase in size (serializing blocks with great amounts
of transactions).

As a result of this, some readability is sacrificed (it's quite a ways around the standard library methods for 
serializing binary data), and consequently the usage is a little bit different from Go standards. However I believe the 
efficiency tradeoff here is definitely worth the workaround, especially so when the network increases in size.

##Methods

###Integers (integers.go)

####Standard

Functions are provided for integers of any size (between 8 and 64 bit). All integers will need to be passed and 
retrieved as their unsigned format. Signed integers can be converted to and from their unsigned format by simple type 
conversion. Integer serialization makes use of a free list of buffers.

The functions are declared like this:

**Reading**
- `Uint8(r io.Reader) (uint8, error)`
- `Uint16(r io.Reader, o binary.ByteOrder) (uint16, error)`
- `Uint32(r io.Reader, o binary.ByteOrder) (uint32, error)`
- `Uint64(r io.Reader, o binary.ByteOrder) (uint64, error)`

**Writing**
- `PutUint8(w io.Writer, v uint8) error`
- `PutUint16(w io.Writer, o binary.ByteOrder, v uint16) error`
- `PutUint32(w io.Writer, o binary.ByteOrder, v uint32) error`
- `PutUint64(w io.Writer, o binary.ByteOrder, v uint64) error`

Usage of this module will be as follows. Let's take a uint64 for example:

```go
var x uint64 = 5

buf := new(bytes.Buffer)

err := encoding.PutUint64(buf, binary.LittleEndian, x)
// Handle error

// buf.Bytes() = [5 0 0 0 0 0 0 0]

y, err := encoding.Uint64(buf, binary.LittleEndian)
// Handle error

// y = 5
```

For an int64 you would write:

```go
var x int64 = -5

buf := new(bytes.Buffer)

err := encoding.PutUint64(buf, binary.LittleEndian, uint64(x))
// Handle error

// buf.Bytes() = [251 255 255 255 255 255 255 255]

uy, err := encoding.Uint64(buf, binary.LittleEndian)
// Handle error

y := int64(uy)

// y = -5
```

####CompactSize

The library provides an implementation of CompactSize integers as described in the 
[Bitcoin Developer Reference](https://bitcoin.org/en/developer-reference#compactsize-unsigned-integers).
The functions implemented are as follows:
- `ReadVarInt(r io.Reader) (uint64, error)` for reading CompactSize integers
- `WriteVarInt(w io.Writer, v uint64) error` for writing CompactSize integers
- `VarIntSerializeSize(v uint64) error` for determining how many bytes an integer will take up through CompactSize
encoding

For reading and writing CompactSize integers, the functions make use of the integer serialization functions.

###Hashes (hashes.go)

As hashes are stored as byte slices of length 32, they get a seperate source file to provide a free list of buffers.
The functions are quite straightforward.
- `ReadHash(r io.Reader) ([]byte, error)` to read a hash
- `WriteHash(w io.Writer, b []byte) error` to write a hash

###Strings and byte slices (vardata.go)

Functions in this source file will make use of CompactSize integers for embedding and reading length prefixes.
For byte slices of variable length (like a signature) there are two functions.

- `ReadVarBytes(r io.Reader) ([]byte, error)` to read byte data
- `WriteVarBytes(w io.Writer, b []byte) error` to write byte data

For strings of variable length (like a message) there are two convenience functions, which point to the functions
above.

- `ReadString(r io.Reader) (string, error)`
- `WriteString(w io.Writer, s string) error`

Writing a string will simply convert it to a byte slice, and reading it will take the byte slice and convert it
to a string.

###Arrays/slices

For arrays or slices, you should use a CompactSize integer to denote the length of the array, followed by the data,
serialized in a for loop.

The method is as follows (I will take a slice of strings for this example):

```go
arr := []string{"foo", "bar", "baz"}

buf := new(bytes.Buffer)
err := encoding.WriteVarInt(buf, uint64(len(arr)))
// Handle error

for _, str := range arr {
	err := encoding.WriteString(buf, str)
	// Handle error
}

// buf.Bytes() = [3 3 102 111 111 3 98 97 114 3 98 97 122]

count, err := ReadVarInt(buf)
// Handle error

// Make new array with len count
newArr := make([]string, count)
for i := uint64(0); i < count; i++ {
	newArr[i], err := encoding.ReadString(buf)
	// Handle error
}

// newArr = ["foo" "bar" "baz"]
```

###Structs

For structs, the best way of doing this is to create an `Encode()` and `Decode()` method. In these functions you should
go off each field one by one, and encode them into the buffer, or decode them one by one from the buffer, and then 
populate the respective fields.

A more advanced example can be found in the transactions folder. I will demonstrate a small one here:

```go
type S struct {
	Str string
	A []uint64
	B bool
}

func (s *S) Encode(w io.Writer) error {
	// Str
	err := WriteString(w, s.Str)
	// Handle error
	
	// A
	err := WriteVarInt(w, uint64(len(s.A)))
	// Handle error
	for _, n := range s.A {
		err := PutUint64(w, binary.LittleEndian, n)
		// Handle error
	}
	
	// B
	var b uint8
    if s.B == false {
        b = 0
    } else {
        b = 1
    }
	err := PutUint8(w, b)
	return err
}

func (s *S) Decode(r io.Reader) error {
	// Str
	str, err := ReadString(r)
	// Handle error
	s.Str = str
	
	// A
	count, err := ReadVarInt(r)
	// Handle error
	
	s.A = make([]uint64, count)
	for i := uint64(0); i < length; i++ {
		n, err := Uint64(r, binary.LittleEndian)
		// Handle error
		
		// Populate
		s.A[i] = n
	}
	
	// B
	var b bool
	n, err := Uint8(r)
	if n == 0 {
		b = false
	} else {
		b = true
	}
	s.B = b
	
    return nil
}
```

As a sidenote for structs, it's advisable to include a function like `GetEncodeSize()` as seen in the stealthtx source
file. Knowing how long the buffer has to be before running `Encode()` will reduce allocations during encoding, and
increase performance (especially with transactions and blocks as they will contain many different fields).

```go
count := StealthTX.GetEncodeSize()
bs := make([]byte, 0, count)
buf := bytes.NewBuffer(bs)
err := StealthTX.Encode(buf)
// etc.
```