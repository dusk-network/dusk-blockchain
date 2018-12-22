package bytesutil

import "bytes"

// Append takes a buffer and a variadic byte slice
// then appends all elements into the buffer
func Append(buf *bytes.Buffer, data ...[]byte) *bytes.Buffer {

	for _, b := range data {
		buf.Write(b)
	}
	return buf
}
