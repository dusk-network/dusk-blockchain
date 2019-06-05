package processing

import "bytes"

// Encode a slice of bytes to a null-terminated frame
func Encode(p *bytes.Buffer) *bytes.Buffer {
	buf := new(bytes.Buffer)
	buf.Grow(p.Len())
	writeBlock := func(p []byte) {
		buf.WriteByte(byte(len(p) + 1))
		buf.Write(p)
	}
	for _, ch := range bytes.Split(p.Bytes(), []byte{0}) {
		for len(ch) > 0xfe {
			writeBlock(ch[:0xfe])
			ch = ch[0xfe:]
		}
		writeBlock(ch)
	}
	buf.WriteByte(0)
	return buf
}

// Decode a null-terminated frame to a slice of bytes
func Decode(p []byte) *bytes.Buffer {
	if len(p) == 0 {
		return nil
	}
	buf := new(bytes.Buffer)
	buf.Grow(len(p))
	for n := p[0]; n > 0; n = p[0] {
		if int(n) >= len(p) {
			return nil
		}
		buf.Write(p[1:n])
		p = p[n:]
		if n < 0xff && p[0] > 0 {
			buf.WriteByte(0)
		}
	}
	return buf
}
