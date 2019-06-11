package processing

import (
	"bytes"
)

// EncodedSize calculates size of encoded message
func EncodedSize(n int) int {
	return n + n/256
}

// Encode a null-terminated slice of bytes to a cobs frame
func Encode(b *bytes.Buffer) *bytes.Buffer {
	p := b.Bytes()
	if len(p) == 0 {
		return new(bytes.Buffer)
	}
	// pad inital message with zero, if missing
	if p[len(p)-1] != 0 {
		p = append(p, 0)
	}
	var buf bytes.Buffer
	for {
		i := bytes.IndexByte(p, 0)
		// no more zeros, we are done
		if i < 0 {
			buf.WriteByte(0)
			return &buf
		}
		// split oversized chunks
		for i >= 254 {
			buf.WriteByte(255)
			buf.Write(p[:254])
			p = p[254:]
			i -= 254
		}
		// write rest of the chunk
		buf.WriteByte(byte(i + 1))
		buf.Write(p[:i])
		p = p[i+1:]
	}
}

// Decode a cobs frame to a null-terminated slice of bytes
func Decode(p []byte) *bytes.Buffer {
	if len(p) == 0 {
		return new(bytes.Buffer)
	}
	// Cut delimiter
	p = p[:len(p)-1]
	var buf bytes.Buffer
	for {
		// nothing left, we are done
		if len(p) == 0 {
			return &buf
		}
		n, body := p[0], p[1:]
		// invalid frame, abort
		if int(n-1) > len(body) || n == 0 {
			return nil
		}
		buf.Write(body[:n-1])
		// full blocks are not followed by zero
		if n < 255 {
			buf.WriteByte(0)
		}
		p = p[n:]
	}
}
