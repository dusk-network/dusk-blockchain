package ristretto

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"

	// Requires for FieldElement.[Set]BigInt().  Obviously not used for actual
	// implementation, as operations on big.Ints are  not constant-time.
	"math/big"
)

// A number modulo the prime l, where l is the order of the Ristretto group
// over Edwards25519.
//
// The scalar s is represented as an array s[0], ... s[31] with 0 <= s[i] <= 255
// and s = s[0] + s[1] * 256 + s[2] * 65536 + ... + s[31] * 256^31.
type Scalar [32]byte

var (
	scZero Scalar
	scOne  = Scalar{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	scL = Scalar{
		0xED, 0xD3, 0xF5, 0x5C, 0x1A, 0x63, 0x12, 0x58, 0xD6, 0x9C, 0xF7,
		0xA2, 0xDE, 0xF9, 0xDE, 0x14, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
	}
)

// Encode s little endian into buf. Returns s.
func (s *Scalar) BytesInto(buf *[32]byte) *Scalar {
	copy(buf[:], s[:])
	return s
}

// Bytes() returns a little-endian packed version of s.  See also BytesInto().
func (s *Scalar) Bytes() []byte {
	var ret [32]byte
	s.BytesInto(&ret)
	return ret[:]
}

// Sets s to x mod l, where x is interpreted little endian and the
// top 3 bits are ignored.  Returns s.
func (s *Scalar) SetBytes(x *[32]byte) *Scalar {
	copy(s[:], x[:])
	s[31] &= 0x1f
	return s.Sub(s, &scL)
}

// Sets s to -a.  Returns s.
func (s *Scalar) Neg(a *Scalar) *Scalar {
	return s.Sub(&scZero, a)
}

// Sets s to a + b.  Returns s.
func (s *Scalar) Add(a, b *Scalar) *Scalar {
	var carry uint16
	for i := 0; i < 32; i++ {
		carry = uint16(a[i]) + uint16(b[i]) + (carry >> 8)
		s[i] = uint8(carry)
	}
	return s.Sub(s, &scL)
}

// Sets s to a - b.  Returns s.
func (s *Scalar) Sub(a, b *Scalar) *Scalar {
	var borrow uint16
	for i := 0; i < 32; i++ {
		borrow = uint16(a[i]) - uint16(b[i]) - (borrow >> 15)
		s[i] = uint8(borrow)
	}

	// Add l if underflown
	ufMask := ((borrow >> 15) ^ 1) - 1
	var carry uint16
	for i := 0; i < 32; i++ {
		carry = uint16(s[i]) + (carry >> 8) + (uint16(scL[i]) & ufMask)
		s[i] = uint8(carry)
	}
	return s
}

// Returns s as a big.Int.
//
// Warning: operations on big.Ints are not constant-time: do not use them
// for cryptography unless you're sure this is not an issue.
func (s *Scalar) BigInt() *big.Int {
	var ret big.Int
	var buf, rBuf [32]byte
	s.BytesInto(&buf)
	for i := 0; i < 32; i++ {
		rBuf[i] = buf[31-i]
	}
	return ret.SetBytes(rBuf[:])
}

// Sets s to x modulo l.
//
// Warning: operations on big.Ints are not constant-time: do not use them
// for cryptography unless you're sure this is not an issue.
func (s *Scalar) SetBigInt(x *big.Int) *Scalar {
	var v, biL big.Int
	biL.SetString(
		"1000000000000000000000000000000014def9dea2f79cd65812631a5cf5d3ed", 16)
	buf := v.Mod(x, &biL).Bytes()
	var rBuf [32]byte
	for i := 0; i < len(buf) && i < 32; i++ {
		rBuf[i] = buf[len(buf)-i-1]
	}
	*s = rBuf
	return s
}

// Sets s to t.  Returns s.
func (s *Scalar) Set(t *Scalar) *Scalar {
	copy(s[:], t[:])
	return s
}

// Sets s to 0.  Returns s.
func (s *Scalar) SetZero() *Scalar {
	return s.Set(&scZero)
}

// Sets s to 1.  Returns s.
func (s *Scalar) SetOne() *Scalar {
	return s.Set(&scOne)
}

// Interprets a 3-byte unsigned little endian byte-slice as int64
func load3(in []byte) int64 {
	var r int64
	r = int64(in[0])
	r |= int64(in[1]) << 8
	r |= int64(in[2]) << 16
	return r
}

// Interprets a 4-byte unsigned little endian byte-slice as int64
func load4(in []byte) int64 {
	var r int64
	r = int64(in[0])
	r |= int64(in[1]) << 8
	r |= int64(in[2]) << 16
	r |= int64(in[3]) << 24
	return r
}

// Sets s to a * b + c.  Returns s.
func (s *Scalar) MulAdd(a, b, c *Scalar) *Scalar {
	a0 := 0x1FFFFF & load3(a[:])
	a1 := 0x1FFFFF & (load4(a[2:]) >> 5)
	a2 := 0x1FFFFF & (load3(a[5:]) >> 2)
	a3 := 0x1FFFFF & (load4(a[7:]) >> 7)
	a4 := 0x1FFFFF & (load4(a[10:]) >> 4)
	a5 := 0x1FFFFF & (load3(a[13:]) >> 1)
	a6 := 0x1FFFFF & (load4(a[15:]) >> 6)
	a7 := 0x1FFFFF & (load3(a[18:]) >> 3)
	a8 := 0x1FFFFF & load3(a[21:])
	a9 := 0x1FFFFF & (load4(a[23:]) >> 5)
	a10 := 0x1FFFFF & (load3(a[26:]) >> 2)
	a11 := (load4(a[28:]) >> 7)
	b0 := 0x1FFFFF & load3(b[:])
	b1 := 0x1FFFFF & (load4(b[2:]) >> 5)
	b2 := 0x1FFFFF & (load3(b[5:]) >> 2)
	b3 := 0x1FFFFF & (load4(b[7:]) >> 7)
	b4 := 0x1FFFFF & (load4(b[10:]) >> 4)
	b5 := 0x1FFFFF & (load3(b[13:]) >> 1)
	b6 := 0x1FFFFF & (load4(b[15:]) >> 6)
	b7 := 0x1FFFFF & (load3(b[18:]) >> 3)
	b8 := 0x1FFFFF & load3(b[21:])
	b9 := 0x1FFFFF & (load4(b[23:]) >> 5)
	b10 := 0x1FFFFF & (load3(b[26:]) >> 2)
	b11 := (load4(b[28:]) >> 7)
	c0 := 0x1FFFFF & load3(c[:])
	c1 := 0x1FFFFF & (load4(c[2:]) >> 5)
	c2 := 0x1FFFFF & (load3(c[5:]) >> 2)
	c3 := 0x1FFFFF & (load4(c[7:]) >> 7)
	c4 := 0x1FFFFF & (load4(c[10:]) >> 4)
	c5 := 0x1FFFFF & (load3(c[13:]) >> 1)
	c6 := 0x1FFFFF & (load4(c[15:]) >> 6)
	c7 := 0x1FFFFF & (load3(c[18:]) >> 3)
	c8 := 0x1FFFFF & load3(c[21:])
	c9 := 0x1FFFFF & (load4(c[23:]) >> 5)
	c10 := 0x1FFFFF & (load3(c[26:]) >> 2)
	c11 := (load4(c[28:]) >> 7)
	var carry [23]int64

	s0 := c0 + a0*b0
	s1 := c1 + a0*b1 + a1*b0
	s2 := c2 + a0*b2 + a1*b1 + a2*b0
	s3 := c3 + a0*b3 + a1*b2 + a2*b1 + a3*b0
	s4 := c4 + a0*b4 + a1*b3 + a2*b2 + a3*b1 + a4*b0
	s5 := c5 + a0*b5 + a1*b4 + a2*b3 + a3*b2 + a4*b1 + a5*b0
	s6 := c6 + a0*b6 + a1*b5 + a2*b4 + a3*b3 + a4*b2 + a5*b1 + a6*b0
	s7 := c7 + a0*b7 + a1*b6 + a2*b5 + a3*b4 + a4*b3 + a5*b2 + a6*b1 + a7*b0
	s8 := c8 + a0*b8 + a1*b7 + a2*b6 + a3*b5 + a4*b4 + a5*b3 + a6*b2 + a7*b1 + a8*b0
	s9 := c9 + a0*b9 + a1*b8 + a2*b7 + a3*b6 + a4*b5 + a5*b4 + a6*b3 + a7*b2 + a8*b1 + a9*b0
	s10 := c10 + a0*b10 + a1*b9 + a2*b8 + a3*b7 + a4*b6 + a5*b5 + a6*b4 + a7*b3 + a8*b2 + a9*b1 + a10*b0
	s11 := c11 + a0*b11 + a1*b10 + a2*b9 + a3*b8 + a4*b7 + a5*b6 + a6*b5 + a7*b4 + a8*b3 + a9*b2 + a10*b1 + a11*b0
	s12 := a1*b11 + a2*b10 + a3*b9 + a4*b8 + a5*b7 + a6*b6 + a7*b5 + a8*b4 + a9*b3 + a10*b2 + a11*b1
	s13 := a2*b11 + a3*b10 + a4*b9 + a5*b8 + a6*b7 + a7*b6 + a8*b5 + a9*b4 + a10*b3 + a11*b2
	s14 := a3*b11 + a4*b10 + a5*b9 + a6*b8 + a7*b7 + a8*b6 + a9*b5 + a10*b4 + a11*b3
	s15 := a4*b11 + a5*b10 + a6*b9 + a7*b8 + a8*b7 + a9*b6 + a10*b5 + a11*b4
	s16 := a5*b11 + a6*b10 + a7*b9 + a8*b8 + a9*b7 + a10*b6 + a11*b5
	s17 := a6*b11 + a7*b10 + a8*b9 + a9*b8 + a10*b7 + a11*b6
	s18 := a7*b11 + a8*b10 + a9*b9 + a10*b8 + a11*b7
	s19 := a8*b11 + a9*b10 + a10*b9 + a11*b8
	s20 := a9*b11 + a10*b10 + a11*b9
	s21 := a10*b11 + a11*b10
	s22 := a11 * b11
	s23 := int64(0)

	carry[0] = (s0 + (1 << 20)) >> 21
	s1 += carry[0]
	s0 -= carry[0] << 21
	carry[2] = (s2 + (1 << 20)) >> 21
	s3 += carry[2]
	s2 -= carry[2] << 21
	carry[4] = (s4 + (1 << 20)) >> 21
	s5 += carry[4]
	s4 -= carry[4] << 21
	carry[6] = (s6 + (1 << 20)) >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[8] = (s8 + (1 << 20)) >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[10] = (s10 + (1 << 20)) >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21
	carry[12] = (s12 + (1 << 20)) >> 21
	s13 += carry[12]
	s12 -= carry[12] << 21
	carry[14] = (s14 + (1 << 20)) >> 21
	s15 += carry[14]
	s14 -= carry[14] << 21
	carry[16] = (s16 + (1 << 20)) >> 21
	s17 += carry[16]
	s16 -= carry[16] << 21
	carry[18] = (s18 + (1 << 20)) >> 21
	s19 += carry[18]
	s18 -= carry[18] << 21
	carry[20] = (s20 + (1 << 20)) >> 21
	s21 += carry[20]
	s20 -= carry[20] << 21
	carry[22] = (s22 + (1 << 20)) >> 21
	s23 += carry[22]
	s22 -= carry[22] << 21

	carry[1] = (s1 + (1 << 20)) >> 21
	s2 += carry[1]
	s1 -= carry[1] << 21
	carry[3] = (s3 + (1 << 20)) >> 21
	s4 += carry[3]
	s3 -= carry[3] << 21
	carry[5] = (s5 + (1 << 20)) >> 21
	s6 += carry[5]
	s5 -= carry[5] << 21
	carry[7] = (s7 + (1 << 20)) >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[9] = (s9 + (1 << 20)) >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[11] = (s11 + (1 << 20)) >> 21
	s12 += carry[11]
	s11 -= carry[11] << 21
	carry[13] = (s13 + (1 << 20)) >> 21
	s14 += carry[13]
	s13 -= carry[13] << 21
	carry[15] = (s15 + (1 << 20)) >> 21
	s16 += carry[15]
	s15 -= carry[15] << 21
	carry[17] = (s17 + (1 << 20)) >> 21
	s18 += carry[17]
	s17 -= carry[17] << 21
	carry[19] = (s19 + (1 << 20)) >> 21
	s20 += carry[19]
	s19 -= carry[19] << 21
	carry[21] = (s21 + (1 << 20)) >> 21
	s22 += carry[21]
	s21 -= carry[21] << 21

	s11 += s23 * 666643
	s12 += s23 * 470296
	s13 += s23 * 654183
	s14 -= s23 * 997805
	s15 += s23 * 136657
	s16 -= s23 * 683901
	s23 = 0

	s10 += s22 * 666643
	s11 += s22 * 470296
	s12 += s22 * 654183
	s13 -= s22 * 997805
	s14 += s22 * 136657
	s15 -= s22 * 683901
	s22 = 0

	s9 += s21 * 666643
	s10 += s21 * 470296
	s11 += s21 * 654183
	s12 -= s21 * 997805
	s13 += s21 * 136657
	s14 -= s21 * 683901
	s21 = 0

	s8 += s20 * 666643
	s9 += s20 * 470296
	s10 += s20 * 654183
	s11 -= s20 * 997805
	s12 += s20 * 136657
	s13 -= s20 * 683901
	s20 = 0

	s7 += s19 * 666643
	s8 += s19 * 470296
	s9 += s19 * 654183
	s10 -= s19 * 997805
	s11 += s19 * 136657
	s12 -= s19 * 683901
	s19 = 0

	s6 += s18 * 666643
	s7 += s18 * 470296
	s8 += s18 * 654183
	s9 -= s18 * 997805
	s10 += s18 * 136657
	s11 -= s18 * 683901
	s18 = 0

	carry[6] = (s6 + (1 << 20)) >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[8] = (s8 + (1 << 20)) >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[10] = (s10 + (1 << 20)) >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21
	carry[12] = (s12 + (1 << 20)) >> 21
	s13 += carry[12]
	s12 -= carry[12] << 21
	carry[14] = (s14 + (1 << 20)) >> 21
	s15 += carry[14]
	s14 -= carry[14] << 21
	carry[16] = (s16 + (1 << 20)) >> 21
	s17 += carry[16]
	s16 -= carry[16] << 21

	carry[7] = (s7 + (1 << 20)) >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[9] = (s9 + (1 << 20)) >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[11] = (s11 + (1 << 20)) >> 21
	s12 += carry[11]
	s11 -= carry[11] << 21
	carry[13] = (s13 + (1 << 20)) >> 21
	s14 += carry[13]
	s13 -= carry[13] << 21
	carry[15] = (s15 + (1 << 20)) >> 21
	s16 += carry[15]
	s15 -= carry[15] << 21

	s5 += s17 * 666643
	s6 += s17 * 470296
	s7 += s17 * 654183
	s8 -= s17 * 997805
	s9 += s17 * 136657
	s10 -= s17 * 683901
	s17 = 0

	s4 += s16 * 666643
	s5 += s16 * 470296
	s6 += s16 * 654183
	s7 -= s16 * 997805
	s8 += s16 * 136657
	s9 -= s16 * 683901
	s16 = 0

	s3 += s15 * 666643
	s4 += s15 * 470296
	s5 += s15 * 654183
	s6 -= s15 * 997805
	s7 += s15 * 136657
	s8 -= s15 * 683901
	s15 = 0

	s2 += s14 * 666643
	s3 += s14 * 470296
	s4 += s14 * 654183
	s5 -= s14 * 997805
	s6 += s14 * 136657
	s7 -= s14 * 683901
	s14 = 0

	s1 += s13 * 666643
	s2 += s13 * 470296
	s3 += s13 * 654183
	s4 -= s13 * 997805
	s5 += s13 * 136657
	s6 -= s13 * 683901
	s13 = 0

	s0 += s12 * 666643
	s1 += s12 * 470296
	s2 += s12 * 654183
	s3 -= s12 * 997805
	s4 += s12 * 136657
	s5 -= s12 * 683901
	s12 = 0

	carry[0] = (s0 + (1 << 20)) >> 21
	s1 += carry[0]
	s0 -= carry[0] << 21
	carry[2] = (s2 + (1 << 20)) >> 21
	s3 += carry[2]
	s2 -= carry[2] << 21
	carry[4] = (s4 + (1 << 20)) >> 21
	s5 += carry[4]
	s4 -= carry[4] << 21
	carry[6] = (s6 + (1 << 20)) >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[8] = (s8 + (1 << 20)) >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[10] = (s10 + (1 << 20)) >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21

	carry[1] = (s1 + (1 << 20)) >> 21
	s2 += carry[1]
	s1 -= carry[1] << 21
	carry[3] = (s3 + (1 << 20)) >> 21
	s4 += carry[3]
	s3 -= carry[3] << 21
	carry[5] = (s5 + (1 << 20)) >> 21
	s6 += carry[5]
	s5 -= carry[5] << 21
	carry[7] = (s7 + (1 << 20)) >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[9] = (s9 + (1 << 20)) >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[11] = (s11 + (1 << 20)) >> 21
	s12 += carry[11]
	s11 -= carry[11] << 21

	s0 += s12 * 666643
	s1 += s12 * 470296
	s2 += s12 * 654183
	s3 -= s12 * 997805
	s4 += s12 * 136657
	s5 -= s12 * 683901
	s12 = 0

	carry[0] = s0 >> 21
	s1 += carry[0]
	s0 -= carry[0] << 21
	carry[1] = s1 >> 21
	s2 += carry[1]
	s1 -= carry[1] << 21
	carry[2] = s2 >> 21
	s3 += carry[2]
	s2 -= carry[2] << 21
	carry[3] = s3 >> 21
	s4 += carry[3]
	s3 -= carry[3] << 21
	carry[4] = s4 >> 21
	s5 += carry[4]
	s4 -= carry[4] << 21
	carry[5] = s5 >> 21
	s6 += carry[5]
	s5 -= carry[5] << 21
	carry[6] = s6 >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[7] = s7 >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[8] = s8 >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[9] = s9 >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[10] = s10 >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21
	carry[11] = s11 >> 21
	s12 += carry[11]
	s11 -= carry[11] << 21

	s0 += s12 * 666643
	s1 += s12 * 470296
	s2 += s12 * 654183
	s3 -= s12 * 997805
	s4 += s12 * 136657
	s5 -= s12 * 683901
	s12 = 0

	carry[0] = s0 >> 21
	s1 += carry[0]
	s0 -= carry[0] << 21
	carry[1] = s1 >> 21
	s2 += carry[1]
	s1 -= carry[1] << 21
	carry[2] = s2 >> 21
	s3 += carry[2]
	s2 -= carry[2] << 21
	carry[3] = s3 >> 21
	s4 += carry[3]
	s3 -= carry[3] << 21
	carry[4] = s4 >> 21
	s5 += carry[4]
	s4 -= carry[4] << 21
	carry[5] = s5 >> 21
	s6 += carry[5]
	s5 -= carry[5] << 21
	carry[6] = s6 >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[7] = s7 >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[8] = s8 >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[9] = s9 >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[10] = s10 >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21

	s[0] = byte(s0 >> 0)
	s[1] = byte(s0 >> 8)
	s[2] = byte((s0 >> 16) | (s1 << 5))
	s[3] = byte(s1 >> 3)
	s[4] = byte(s1 >> 11)
	s[5] = byte((s1 >> 19) | (s2 << 2))
	s[6] = byte(s2 >> 6)
	s[7] = byte((s2 >> 14) | (s3 << 7))
	s[8] = byte(s3 >> 1)
	s[9] = byte(s3 >> 9)
	s[10] = byte((s3 >> 17) | (s4 << 4))
	s[11] = byte(s4 >> 4)
	s[12] = byte(s4 >> 12)
	s[13] = byte((s4 >> 20) | (s5 << 1))
	s[14] = byte(s5 >> 7)
	s[15] = byte((s5 >> 15) | (s6 << 6))
	s[16] = byte(s6 >> 2)
	s[17] = byte(s6 >> 10)
	s[18] = byte((s6 >> 18) | (s7 << 3))
	s[19] = byte(s7 >> 5)
	s[20] = byte(s7 >> 13)
	s[21] = byte(s8 >> 0)
	s[22] = byte(s8 >> 8)
	s[23] = byte((s8 >> 16) | (s9 << 5))
	s[24] = byte(s9 >> 3)
	s[25] = byte(s9 >> 11)
	s[26] = byte((s9 >> 19) | (s10 << 2))
	s[27] = byte(s10 >> 6)
	s[28] = byte((s10 >> 14) | (s11 << 7))
	s[29] = byte(s11 >> 1)
	s[30] = byte(s11 >> 9)
	s[31] = byte(s11 >> 17)
	return s
}

// Derive sets s to the scalar derived from the given buffer using SHA512 and
// Scalar.SetReduced()  Returns s.
func (s *Scalar) Derive(buf []byte) *Scalar {
	var sBuf [64]byte
	h := sha512.Sum512(buf)
	copy(sBuf[:], h[:])
	return s.SetReduced(&sBuf)
}

// Sets s to t mod l, where t is interpreted little endian.  Returns s.
func (s *Scalar) SetReduced(t *[64]byte) *Scalar {
	t0 := 0x1FFFFF & load3(t[:])
	t1 := 0x1FFFFF & (load4(t[2:]) >> 5)
	t2 := 0x1FFFFF & (load3(t[5:]) >> 2)
	t3 := 0x1FFFFF & (load4(t[7:]) >> 7)
	t4 := 0x1FFFFF & (load4(t[10:]) >> 4)
	t5 := 0x1FFFFF & (load3(t[13:]) >> 1)
	t6 := 0x1FFFFF & (load4(t[15:]) >> 6)
	t7 := 0x1FFFFF & (load3(t[18:]) >> 3)
	t8 := 0x1FFFFF & load3(t[21:])
	t9 := 0x1FFFFF & (load4(t[23:]) >> 5)
	t10 := 0x1FFFFF & (load3(t[26:]) >> 2)
	t11 := 0x1FFFFF & (load4(t[28:]) >> 7)
	t12 := 0x1FFFFF & (load4(t[31:]) >> 4)
	t13 := 0x1FFFFF & (load3(t[34:]) >> 1)
	t14 := 0x1FFFFF & (load4(t[36:]) >> 6)
	t15 := 0x1FFFFF & (load3(t[39:]) >> 3)
	t16 := 0x1FFFFF & load3(t[42:])
	t17 := 0x1FFFFF & (load4(t[44:]) >> 5)
	t18 := 0x1FFFFF & (load3(t[47:]) >> 2)
	t19 := 0x1FFFFF & (load4(t[49:]) >> 7)
	t20 := 0x1FFFFF & (load4(t[52:]) >> 4)
	t21 := 0x1FFFFF & (load3(t[55:]) >> 1)
	t22 := 0x1FFFFF & (load4(t[57:]) >> 6)
	t23 := (load4(t[60:]) >> 3)

	t11 += t23 * 666643
	t12 += t23 * 470296
	t13 += t23 * 654183
	t14 -= t23 * 997805
	t15 += t23 * 136657
	t16 -= t23 * 683901
	t23 = 0

	t10 += t22 * 666643
	t11 += t22 * 470296
	t12 += t22 * 654183
	t13 -= t22 * 997805
	t14 += t22 * 136657
	t15 -= t22 * 683901
	t22 = 0

	t9 += t21 * 666643
	t10 += t21 * 470296
	t11 += t21 * 654183
	t12 -= t21 * 997805
	t13 += t21 * 136657
	t14 -= t21 * 683901
	t21 = 0

	t8 += t20 * 666643
	t9 += t20 * 470296
	t10 += t20 * 654183
	t11 -= t20 * 997805
	t12 += t20 * 136657
	t13 -= t20 * 683901
	t20 = 0

	t7 += t19 * 666643
	t8 += t19 * 470296
	t9 += t19 * 654183
	t10 -= t19 * 997805
	t11 += t19 * 136657
	t12 -= t19 * 683901
	t19 = 0

	t6 += t18 * 666643
	t7 += t18 * 470296
	t8 += t18 * 654183
	t9 -= t18 * 997805
	t10 += t18 * 136657
	t11 -= t18 * 683901
	t18 = 0

	var carry [17]int64

	carry[6] = (t6 + (1 << 20)) >> 21
	t7 += carry[6]
	t6 -= carry[6] << 21
	carry[8] = (t8 + (1 << 20)) >> 21
	t9 += carry[8]
	t8 -= carry[8] << 21
	carry[10] = (t10 + (1 << 20)) >> 21
	t11 += carry[10]
	t10 -= carry[10] << 21
	carry[12] = (t12 + (1 << 20)) >> 21
	t13 += carry[12]
	t12 -= carry[12] << 21
	carry[14] = (t14 + (1 << 20)) >> 21
	t15 += carry[14]
	t14 -= carry[14] << 21
	carry[16] = (t16 + (1 << 20)) >> 21
	t17 += carry[16]
	t16 -= carry[16] << 21

	carry[7] = (t7 + (1 << 20)) >> 21
	t8 += carry[7]
	t7 -= carry[7] << 21
	carry[9] = (t9 + (1 << 20)) >> 21
	t10 += carry[9]
	t9 -= carry[9] << 21
	carry[11] = (t11 + (1 << 20)) >> 21
	t12 += carry[11]
	t11 -= carry[11] << 21
	carry[13] = (t13 + (1 << 20)) >> 21
	t14 += carry[13]
	t13 -= carry[13] << 21
	carry[15] = (t15 + (1 << 20)) >> 21
	t16 += carry[15]
	t15 -= carry[15] << 21

	t5 += t17 * 666643
	t6 += t17 * 470296
	t7 += t17 * 654183
	t8 -= t17 * 997805
	t9 += t17 * 136657
	t10 -= t17 * 683901
	t17 = 0

	t4 += t16 * 666643
	t5 += t16 * 470296
	t6 += t16 * 654183
	t7 -= t16 * 997805
	t8 += t16 * 136657
	t9 -= t16 * 683901
	t16 = 0

	t3 += t15 * 666643
	t4 += t15 * 470296
	t5 += t15 * 654183
	t6 -= t15 * 997805
	t7 += t15 * 136657
	t8 -= t15 * 683901
	t15 = 0

	t2 += t14 * 666643
	t3 += t14 * 470296
	t4 += t14 * 654183
	t5 -= t14 * 997805
	t6 += t14 * 136657
	t7 -= t14 * 683901
	t14 = 0

	t1 += t13 * 666643
	t2 += t13 * 470296
	t3 += t13 * 654183
	t4 -= t13 * 997805
	t5 += t13 * 136657
	t6 -= t13 * 683901
	t13 = 0

	t0 += t12 * 666643
	t1 += t12 * 470296
	t2 += t12 * 654183
	t3 -= t12 * 997805
	t4 += t12 * 136657
	t5 -= t12 * 683901
	t12 = 0

	carry[0] = (t0 + (1 << 20)) >> 21
	t1 += carry[0]
	t0 -= carry[0] << 21
	carry[2] = (t2 + (1 << 20)) >> 21
	t3 += carry[2]
	t2 -= carry[2] << 21
	carry[4] = (t4 + (1 << 20)) >> 21
	t5 += carry[4]
	t4 -= carry[4] << 21
	carry[6] = (t6 + (1 << 20)) >> 21
	t7 += carry[6]
	t6 -= carry[6] << 21
	carry[8] = (t8 + (1 << 20)) >> 21
	t9 += carry[8]
	t8 -= carry[8] << 21
	carry[10] = (t10 + (1 << 20)) >> 21
	t11 += carry[10]
	t10 -= carry[10] << 21

	carry[1] = (t1 + (1 << 20)) >> 21
	t2 += carry[1]
	t1 -= carry[1] << 21
	carry[3] = (t3 + (1 << 20)) >> 21
	t4 += carry[3]
	t3 -= carry[3] << 21
	carry[5] = (t5 + (1 << 20)) >> 21
	t6 += carry[5]
	t5 -= carry[5] << 21
	carry[7] = (t7 + (1 << 20)) >> 21
	t8 += carry[7]
	t7 -= carry[7] << 21
	carry[9] = (t9 + (1 << 20)) >> 21
	t10 += carry[9]
	t9 -= carry[9] << 21
	carry[11] = (t11 + (1 << 20)) >> 21
	t12 += carry[11]
	t11 -= carry[11] << 21

	t0 += t12 * 666643
	t1 += t12 * 470296
	t2 += t12 * 654183
	t3 -= t12 * 997805
	t4 += t12 * 136657
	t5 -= t12 * 683901
	t12 = 0

	carry[0] = t0 >> 21
	t1 += carry[0]
	t0 -= carry[0] << 21
	carry[1] = t1 >> 21
	t2 += carry[1]
	t1 -= carry[1] << 21
	carry[2] = t2 >> 21
	t3 += carry[2]
	t2 -= carry[2] << 21
	carry[3] = t3 >> 21
	t4 += carry[3]
	t3 -= carry[3] << 21
	carry[4] = t4 >> 21
	t5 += carry[4]
	t4 -= carry[4] << 21
	carry[5] = t5 >> 21
	t6 += carry[5]
	t5 -= carry[5] << 21
	carry[6] = t6 >> 21
	t7 += carry[6]
	t6 -= carry[6] << 21
	carry[7] = t7 >> 21
	t8 += carry[7]
	t7 -= carry[7] << 21
	carry[8] = t8 >> 21
	t9 += carry[8]
	t8 -= carry[8] << 21
	carry[9] = t9 >> 21
	t10 += carry[9]
	t9 -= carry[9] << 21
	carry[10] = t10 >> 21
	t11 += carry[10]
	t10 -= carry[10] << 21
	carry[11] = t11 >> 21
	t12 += carry[11]
	t11 -= carry[11] << 21

	t0 += t12 * 666643
	t1 += t12 * 470296
	t2 += t12 * 654183
	t3 -= t12 * 997805
	t4 += t12 * 136657
	t5 -= t12 * 683901
	t12 = 0

	carry[0] = t0 >> 21
	t1 += carry[0]
	t0 -= carry[0] << 21
	carry[1] = t1 >> 21
	t2 += carry[1]
	t1 -= carry[1] << 21
	carry[2] = t2 >> 21
	t3 += carry[2]
	t2 -= carry[2] << 21
	carry[3] = t3 >> 21
	t4 += carry[3]
	t3 -= carry[3] << 21
	carry[4] = t4 >> 21
	t5 += carry[4]
	t4 -= carry[4] << 21
	carry[5] = t5 >> 21
	t6 += carry[5]
	t5 -= carry[5] << 21
	carry[6] = t6 >> 21
	t7 += carry[6]
	t6 -= carry[6] << 21
	carry[7] = t7 >> 21
	t8 += carry[7]
	t7 -= carry[7] << 21
	carry[8] = t8 >> 21
	t9 += carry[8]
	t8 -= carry[8] << 21
	carry[9] = t9 >> 21
	t10 += carry[9]
	t9 -= carry[9] << 21
	carry[10] = t10 >> 21
	t11 += carry[10]
	t10 -= carry[10] << 21

	s[0] = byte(t0 >> 0)
	s[1] = byte(t0 >> 8)
	s[2] = byte((t0 >> 16) | (t1 << 5))
	s[3] = byte(t1 >> 3)
	s[4] = byte(t1 >> 11)
	s[5] = byte((t1 >> 19) | (t2 << 2))
	s[6] = byte(t2 >> 6)
	s[7] = byte((t2 >> 14) | (t3 << 7))
	s[8] = byte(t3 >> 1)
	s[9] = byte(t3 >> 9)
	s[10] = byte((t3 >> 17) | (t4 << 4))
	s[11] = byte(t4 >> 4)
	s[12] = byte(t4 >> 12)
	s[13] = byte((t4 >> 20) | (t5 << 1))
	s[14] = byte(t5 >> 7)
	s[15] = byte((t5 >> 15) | (t6 << 6))
	s[16] = byte(t6 >> 2)
	s[17] = byte(t6 >> 10)
	s[18] = byte((t6 >> 18) | (t7 << 3))
	s[19] = byte(t7 >> 5)
	s[20] = byte(t7 >> 13)
	s[21] = byte(t8 >> 0)
	s[22] = byte(t8 >> 8)
	s[23] = byte((t8 >> 16) | (t9 << 5))
	s[24] = byte(t9 >> 3)
	s[25] = byte(t9 >> 11)
	s[26] = byte((t9 >> 19) | (t10 << 2))
	s[27] = byte(t10 >> 6)
	s[28] = byte((t10 >> 14) | (t11 << 7))
	s[29] = byte(t11 >> 1)
	s[30] = byte(t11 >> 9)
	s[31] = byte(t11 >> 17)

	return s
}

// Sets s to a random scalar.  Returns s.
func (s *Scalar) Rand() *Scalar {
	var buf [64]byte
	rand.Read(buf[:])
	return s.SetReduced(&buf)
}

// SetReduce32 is similar to SetReduced. the input is however 32 bytes and not 64
func (s *Scalar) SetReduce32(t *[32]byte) *Scalar {
	t0 := 2097151 & load3(t[:])
	t1 := 2097151 & (load4(t[2:]) >> 5)
	t2 := 2097151 & (load3(t[5:]) >> 2)
	t3 := 2097151 & (load4(t[7:]) >> 7)
	t4 := 2097151 & (load4(t[10:]) >> 4)
	t5 := 2097151 & (load3(t[13:]) >> 1)
	t6 := 2097151 & (load4(t[15:]) >> 6)
	t7 := 2097151 & (load3(t[18:]) >> 3)
	t8 := 2097151 & load3(t[21:])
	t9 := 2097151 & (load4(t[23:]) >> 5)
	t10 := 2097151 & (load3(t[26:]) >> 2)
	t11 := (load4(t[28:]) >> 7)
	t12 := int64(0)
	var carry [12]int64
	carry[0] = (t0 + (1 << 20)) >> 21
	t1 += carry[0]
	t0 -= carry[0] << 21
	carry[2] = (t2 + (1 << 20)) >> 21
	t3 += carry[2]
	t2 -= carry[2] << 21
	carry[4] = (t4 + (1 << 20)) >> 21
	t5 += carry[4]
	t4 -= carry[4] << 21
	carry[6] = (t6 + (1 << 20)) >> 21
	t7 += carry[6]
	t6 -= carry[6] << 21
	carry[8] = (t8 + (1 << 20)) >> 21
	t9 += carry[8]
	t8 -= carry[8] << 21
	carry[10] = (t10 + (1 << 20)) >> 21
	t11 += carry[10]
	t10 -= carry[10] << 21
	carry[1] = (t1 + (1 << 20)) >> 21
	t2 += carry[1]
	t1 -= carry[1] << 21
	carry[3] = (t3 + (1 << 20)) >> 21
	t4 += carry[3]
	t3 -= carry[3] << 21
	carry[5] = (t5 + (1 << 20)) >> 21
	t6 += carry[5]
	t5 -= carry[5] << 21
	carry[7] = (t7 + (1 << 20)) >> 21
	t8 += carry[7]
	t7 -= carry[7] << 21
	carry[9] = (t9 + (1 << 20)) >> 21
	t10 += carry[9]
	t9 -= carry[9] << 21
	carry[11] = (t11 + (1 << 20)) >> 21
	t12 += carry[11]
	t11 -= carry[11] << 21

	t0 += t12 * 666643
	t1 += t12 * 470296
	t2 += t12 * 654183
	t3 -= t12 * 997805
	t4 += t12 * 136657
	t5 -= t12 * 683901
	t12 = 0

	carry[0] = t0 >> 21
	t1 += carry[0]
	t0 -= carry[0] << 21
	carry[1] = t1 >> 21
	t2 += carry[1]
	t1 -= carry[1] << 21
	carry[2] = t2 >> 21
	t3 += carry[2]
	t2 -= carry[2] << 21
	carry[3] = t3 >> 21
	t4 += carry[3]
	t3 -= carry[3] << 21
	carry[4] = t4 >> 21
	t5 += carry[4]
	t4 -= carry[4] << 21
	carry[5] = t5 >> 21
	t6 += carry[5]
	t5 -= carry[5] << 21
	carry[6] = t6 >> 21
	t7 += carry[6]
	t6 -= carry[6] << 21
	carry[7] = t7 >> 21
	t8 += carry[7]
	t7 -= carry[7] << 21
	carry[8] = t8 >> 21
	t9 += carry[8]
	t8 -= carry[8] << 21
	carry[9] = t9 >> 21
	t10 += carry[9]
	t9 -= carry[9] << 21
	carry[10] = t10 >> 21
	t11 += carry[10]
	t10 -= carry[10] << 21
	carry[11] = t11 >> 21
	t12 += carry[11]
	t11 -= carry[11] << 21

	t0 += t12 * 666643
	t1 += t12 * 470296
	t2 += t12 * 654183
	t3 -= t12 * 997805
	t4 += t12 * 136657
	t5 -= t12 * 683901

	carry[0] = t0 >> 21
	t1 += carry[0]
	t0 -= carry[0] << 21
	carry[1] = t1 >> 21
	t2 += carry[1]
	t1 -= carry[1] << 21
	carry[2] = t2 >> 21
	t3 += carry[2]
	t2 -= carry[2] << 21
	carry[3] = t3 >> 21
	t4 += carry[3]
	t3 -= carry[3] << 21
	carry[4] = t4 >> 21
	t5 += carry[4]
	t4 -= carry[4] << 21
	carry[5] = t5 >> 21
	t6 += carry[5]
	t5 -= carry[5] << 21
	carry[6] = t6 >> 21
	t7 += carry[6]
	t6 -= carry[6] << 21
	carry[7] = t7 >> 21
	t8 += carry[7]
	t7 -= carry[7] << 21
	carry[8] = t8 >> 21
	t9 += carry[8]
	t8 -= carry[8] << 21
	carry[9] = t9 >> 21
	t10 += carry[9]
	t9 -= carry[9] << 21
	carry[10] = t10 >> 21
	t11 += carry[10]
	t10 -= carry[10] << 21

	s[0] = byte(t0 >> 0)
	s[1] = byte(t0 >> 8)
	s[2] = byte((t0 >> 16) | (t1 << 5))
	s[3] = byte(t1 >> 3)
	s[4] = byte(t1 >> 11)
	s[5] = byte((t1 >> 19) | (t2 << 2))
	s[6] = byte(t2 >> 6)
	s[7] = byte((t2 >> 14) | (t3 << 7))
	s[8] = byte(t3 >> 1)
	s[9] = byte(t3 >> 9)
	s[10] = byte((t3 >> 17) | (t4 << 4))
	s[11] = byte(t4 >> 4)
	s[12] = byte(t4 >> 12)
	s[13] = byte((t4 >> 20) | (t5 << 1))
	s[14] = byte(t5 >> 7)
	s[15] = byte((t5 >> 15) | (t6 << 6))
	s[16] = byte(t6 >> 2)
	s[17] = byte(t6 >> 10)
	s[18] = byte((t6 >> 18) | (t7 << 3))
	s[19] = byte(t7 >> 5)
	s[20] = byte(t7 >> 13)
	s[21] = byte(t8 >> 0)
	s[22] = byte(t8 >> 8)
	s[23] = byte((t8 >> 16) | (t9 << 5))
	s[24] = byte(t9 >> 3)
	s[25] = byte(t9 >> 11)
	s[26] = byte((t9 >> 19) | (t10 << 2))
	s[27] = byte(t10 >> 6)
	s[28] = byte((t10 >> 14) | (t11 << 7))
	s[29] = byte(t11 >> 1)
	s[30] = byte(t11 >> 9)
	s[31] = byte(t11 >> 17)
	return s
}

// Sets s to a*a.  Returns s.
func (s *Scalar) Square(a *Scalar) *Scalar {
	a0 := 0x1FFFFF & load3(a[:])
	a1 := 0x1FFFFF & (load4(a[2:]) >> 5)
	a2 := 0x1FFFFF & (load3(a[5:]) >> 2)
	a3 := 0x1FFFFF & (load4(a[7:]) >> 7)
	a4 := 0x1FFFFF & (load4(a[10:]) >> 4)
	a5 := 0x1FFFFF & (load3(a[13:]) >> 1)
	a6 := 0x1FFFFF & (load4(a[15:]) >> 6)
	a7 := 0x1FFFFF & (load3(a[18:]) >> 3)
	a8 := 0x1FFFFF & load3(a[21:])
	a9 := 0x1FFFFF & (load4(a[23:]) >> 5)
	a10 := 0x1FFFFF & (load3(a[26:]) >> 2)
	a11 := (load4(a[28:]) >> 7)
	var carry [23]int64

	s0 := a0 * a0
	s1 := 2 * a0 * a1
	s2 := 2*a0*a2 + a1*a1
	s3 := 2 * (a0*a3 + a1*a2)
	s4 := 2*(a0*a4+a1*a3) + a2*a2
	s5 := 2 * (a0*a5 + a1*a4 + a2*a3)
	s6 := 2*(a0*a6+a1*a5+a2*a4) + a3*a3
	s7 := 2 * (a0*a7 + a1*a6 + a2*a5 + a3*a4)
	s8 := 2*(a0*a8+a1*a7+a2*a6+a3*a5) + a4*a4
	s9 := 2 * (a0*a9 + a1*a8 + a2*a7 + a3*a6 + a4*a5)
	s10 := 2*(a0*a10+a1*a9+a2*a8+a3*a7+a4*a6) + a5*a5
	s11 := 2 * (a0*a11 + a1*a10 + a2*a9 + a3*a8 + a4*a7 + a5*a6)
	s12 := 2*(a1*a11+a2*a10+a3*a9+a4*a8+a5*a7) + a6*a6
	s13 := 2 * (a2*a11 + a3*a10 + a4*a9 + a5*a8 + a6*a7)
	s14 := 2*(a3*a11+a4*a10+a5*a9+a6*a8) + a7*a7
	s15 := 2 * (a4*a11 + a5*a10 + a6*a9 + a7*a8)
	s16 := 2*(a5*a11+a6*a10+a7*a9) + a8*a8
	s17 := 2 * (a6*a11 + a7*a10 + a8*a9)
	s18 := 2*(a7*a11+a8*a10) + a9*a9
	s19 := 2 * (a8*a11 + a9*a10)
	s20 := 2*a9*a11 + a10*a10
	s21 := 2 * a10 * a11
	s22 := a11 * a11
	s23 := int64(0)

	carry[0] = (s0 + (1 << 20)) >> 21
	s1 += carry[0]
	s0 -= carry[0] << 21
	carry[2] = (s2 + (1 << 20)) >> 21
	s3 += carry[2]
	s2 -= carry[2] << 21
	carry[4] = (s4 + (1 << 20)) >> 21
	s5 += carry[4]
	s4 -= carry[4] << 21
	carry[6] = (s6 + (1 << 20)) >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[8] = (s8 + (1 << 20)) >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[10] = (s10 + (1 << 20)) >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21
	carry[12] = (s12 + (1 << 20)) >> 21
	s13 += carry[12]
	s12 -= carry[12] << 21
	carry[14] = (s14 + (1 << 20)) >> 21
	s15 += carry[14]
	s14 -= carry[14] << 21
	carry[16] = (s16 + (1 << 20)) >> 21
	s17 += carry[16]
	s16 -= carry[16] << 21
	carry[18] = (s18 + (1 << 20)) >> 21
	s19 += carry[18]
	s18 -= carry[18] << 21
	carry[20] = (s20 + (1 << 20)) >> 21
	s21 += carry[20]
	s20 -= carry[20] << 21
	carry[22] = (s22 + (1 << 20)) >> 21
	s23 += carry[22]
	s22 -= carry[22] << 21

	carry[1] = (s1 + (1 << 20)) >> 21
	s2 += carry[1]
	s1 -= carry[1] << 21
	carry[3] = (s3 + (1 << 20)) >> 21
	s4 += carry[3]
	s3 -= carry[3] << 21
	carry[5] = (s5 + (1 << 20)) >> 21
	s6 += carry[5]
	s5 -= carry[5] << 21
	carry[7] = (s7 + (1 << 20)) >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[9] = (s9 + (1 << 20)) >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[11] = (s11 + (1 << 20)) >> 21
	s12 += carry[11]
	s11 -= carry[11] << 21
	carry[13] = (s13 + (1 << 20)) >> 21
	s14 += carry[13]
	s13 -= carry[13] << 21
	carry[15] = (s15 + (1 << 20)) >> 21
	s16 += carry[15]
	s15 -= carry[15] << 21
	carry[17] = (s17 + (1 << 20)) >> 21
	s18 += carry[17]
	s17 -= carry[17] << 21
	carry[19] = (s19 + (1 << 20)) >> 21
	s20 += carry[19]
	s19 -= carry[19] << 21
	carry[21] = (s21 + (1 << 20)) >> 21
	s22 += carry[21]
	s21 -= carry[21] << 21

	s11 += s23 * 666643
	s12 += s23 * 470296
	s13 += s23 * 654183
	s14 -= s23 * 997805
	s15 += s23 * 136657
	s16 -= s23 * 683901
	s23 = 0

	s10 += s22 * 666643
	s11 += s22 * 470296
	s12 += s22 * 654183
	s13 -= s22 * 997805
	s14 += s22 * 136657
	s15 -= s22 * 683901
	s22 = 0

	s9 += s21 * 666643
	s10 += s21 * 470296
	s11 += s21 * 654183
	s12 -= s21 * 997805
	s13 += s21 * 136657
	s14 -= s21 * 683901
	s21 = 0

	s8 += s20 * 666643
	s9 += s20 * 470296
	s10 += s20 * 654183
	s11 -= s20 * 997805
	s12 += s20 * 136657
	s13 -= s20 * 683901
	s20 = 0

	s7 += s19 * 666643
	s8 += s19 * 470296
	s9 += s19 * 654183
	s10 -= s19 * 997805
	s11 += s19 * 136657
	s12 -= s19 * 683901
	s19 = 0

	s6 += s18 * 666643
	s7 += s18 * 470296
	s8 += s18 * 654183
	s9 -= s18 * 997805
	s10 += s18 * 136657
	s11 -= s18 * 683901
	s18 = 0

	carry[6] = (s6 + (1 << 20)) >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[8] = (s8 + (1 << 20)) >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[10] = (s10 + (1 << 20)) >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21
	carry[12] = (s12 + (1 << 20)) >> 21
	s13 += carry[12]
	s12 -= carry[12] << 21
	carry[14] = (s14 + (1 << 20)) >> 21
	s15 += carry[14]
	s14 -= carry[14] << 21
	carry[16] = (s16 + (1 << 20)) >> 21
	s17 += carry[16]
	s16 -= carry[16] << 21

	carry[7] = (s7 + (1 << 20)) >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[9] = (s9 + (1 << 20)) >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[11] = (s11 + (1 << 20)) >> 21
	s12 += carry[11]
	s11 -= carry[11] << 21
	carry[13] = (s13 + (1 << 20)) >> 21
	s14 += carry[13]
	s13 -= carry[13] << 21
	carry[15] = (s15 + (1 << 20)) >> 21
	s16 += carry[15]
	s15 -= carry[15] << 21

	s5 += s17 * 666643
	s6 += s17 * 470296
	s7 += s17 * 654183
	s8 -= s17 * 997805
	s9 += s17 * 136657
	s10 -= s17 * 683901
	s17 = 0

	s4 += s16 * 666643
	s5 += s16 * 470296
	s6 += s16 * 654183
	s7 -= s16 * 997805
	s8 += s16 * 136657
	s9 -= s16 * 683901
	s16 = 0

	s3 += s15 * 666643
	s4 += s15 * 470296
	s5 += s15 * 654183
	s6 -= s15 * 997805
	s7 += s15 * 136657
	s8 -= s15 * 683901
	s15 = 0

	s2 += s14 * 666643
	s3 += s14 * 470296
	s4 += s14 * 654183
	s5 -= s14 * 997805
	s6 += s14 * 136657
	s7 -= s14 * 683901
	s14 = 0

	s1 += s13 * 666643
	s2 += s13 * 470296
	s3 += s13 * 654183
	s4 -= s13 * 997805
	s5 += s13 * 136657
	s6 -= s13 * 683901
	s13 = 0

	s0 += s12 * 666643
	s1 += s12 * 470296
	s2 += s12 * 654183
	s3 -= s12 * 997805
	s4 += s12 * 136657
	s5 -= s12 * 683901
	s12 = 0

	carry[0] = (s0 + (1 << 20)) >> 21
	s1 += carry[0]
	s0 -= carry[0] << 21
	carry[2] = (s2 + (1 << 20)) >> 21
	s3 += carry[2]
	s2 -= carry[2] << 21
	carry[4] = (s4 + (1 << 20)) >> 21
	s5 += carry[4]
	s4 -= carry[4] << 21
	carry[6] = (s6 + (1 << 20)) >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[8] = (s8 + (1 << 20)) >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[10] = (s10 + (1 << 20)) >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21

	carry[1] = (s1 + (1 << 20)) >> 21
	s2 += carry[1]
	s1 -= carry[1] << 21
	carry[3] = (s3 + (1 << 20)) >> 21
	s4 += carry[3]
	s3 -= carry[3] << 21
	carry[5] = (s5 + (1 << 20)) >> 21
	s6 += carry[5]
	s5 -= carry[5] << 21
	carry[7] = (s7 + (1 << 20)) >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[9] = (s9 + (1 << 20)) >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[11] = (s11 + (1 << 20)) >> 21
	s12 += carry[11]
	s11 -= carry[11] << 21

	s0 += s12 * 666643
	s1 += s12 * 470296
	s2 += s12 * 654183
	s3 -= s12 * 997805
	s4 += s12 * 136657
	s5 -= s12 * 683901
	s12 = 0

	carry[0] = s0 >> 21
	s1 += carry[0]
	s0 -= carry[0] << 21
	carry[1] = s1 >> 21
	s2 += carry[1]
	s1 -= carry[1] << 21
	carry[2] = s2 >> 21
	s3 += carry[2]
	s2 -= carry[2] << 21
	carry[3] = s3 >> 21
	s4 += carry[3]
	s3 -= carry[3] << 21
	carry[4] = s4 >> 21
	s5 += carry[4]
	s4 -= carry[4] << 21
	carry[5] = s5 >> 21
	s6 += carry[5]
	s5 -= carry[5] << 21
	carry[6] = s6 >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[7] = s7 >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[8] = s8 >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[9] = s9 >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[10] = s10 >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21
	carry[11] = s11 >> 21
	s12 += carry[11]
	s11 -= carry[11] << 21

	s0 += s12 * 666643
	s1 += s12 * 470296
	s2 += s12 * 654183
	s3 -= s12 * 997805
	s4 += s12 * 136657
	s5 -= s12 * 683901
	s12 = 0

	carry[0] = s0 >> 21
	s1 += carry[0]
	s0 -= carry[0] << 21
	carry[1] = s1 >> 21
	s2 += carry[1]
	s1 -= carry[1] << 21
	carry[2] = s2 >> 21
	s3 += carry[2]
	s2 -= carry[2] << 21
	carry[3] = s3 >> 21
	s4 += carry[3]
	s3 -= carry[3] << 21
	carry[4] = s4 >> 21
	s5 += carry[4]
	s4 -= carry[4] << 21
	carry[5] = s5 >> 21
	s6 += carry[5]
	s5 -= carry[5] << 21
	carry[6] = s6 >> 21
	s7 += carry[6]
	s6 -= carry[6] << 21
	carry[7] = s7 >> 21
	s8 += carry[7]
	s7 -= carry[7] << 21
	carry[8] = s8 >> 21
	s9 += carry[8]
	s8 -= carry[8] << 21
	carry[9] = s9 >> 21
	s10 += carry[9]
	s9 -= carry[9] << 21
	carry[10] = s10 >> 21
	s11 += carry[10]
	s10 -= carry[10] << 21

	s[0] = byte(s0 >> 0)
	s[1] = byte(s0 >> 8)
	s[2] = byte((s0 >> 16) | (s1 << 5))
	s[3] = byte(s1 >> 3)
	s[4] = byte(s1 >> 11)
	s[5] = byte((s1 >> 19) | (s2 << 2))
	s[6] = byte(s2 >> 6)
	s[7] = byte((s2 >> 14) | (s3 << 7))
	s[8] = byte(s3 >> 1)
	s[9] = byte(s3 >> 9)
	s[10] = byte((s3 >> 17) | (s4 << 4))
	s[11] = byte(s4 >> 4)
	s[12] = byte(s4 >> 12)
	s[13] = byte((s4 >> 20) | (s5 << 1))
	s[14] = byte(s5 >> 7)
	s[15] = byte((s5 >> 15) | (s6 << 6))
	s[16] = byte(s6 >> 2)
	s[17] = byte(s6 >> 10)
	s[18] = byte((s6 >> 18) | (s7 << 3))
	s[19] = byte(s7 >> 5)
	s[20] = byte(s7 >> 13)
	s[21] = byte(s8 >> 0)
	s[22] = byte(s8 >> 8)
	s[23] = byte((s8 >> 16) | (s9 << 5))
	s[24] = byte(s9 >> 3)
	s[25] = byte(s9 >> 11)
	s[26] = byte((s9 >> 19) | (s10 << 2))
	s[27] = byte(s10 >> 6)
	s[28] = byte((s10 >> 14) | (s11 << 7))
	s[29] = byte(s11 >> 1)
	s[30] = byte(s11 >> 9)
	s[31] = byte(s11 >> 17)
	return s
}

// Sets s to a * b.  Returns s.
func (s *Scalar) Mul(a, b *Scalar) *Scalar {
	return s.MulAdd(a, b, &scZero)
}

// Sets s to 1/t.  Returns s.
func (s *Scalar) Inverse(t *Scalar) *Scalar {
	var t0, t1, t2, t3, t4, t5 Scalar

	t1.Square(t)
	t2.Mul(t, &t1)
	t0.Mul(&t1, &t2)
	t1.Square(&t0)
	t3.Square(&t1)
	t1.Mul(&t2, &t3)
	t2.Square(&t1)
	t3.Mul(&t0, &t2)
	t0.Square(&t3)
	t2.Mul(&t1, &t0)
	t0.Square(&t2)
	t1.Mul(&t2, &t0)
	t0.Square(&t1)
	t1.Mul(&t3, &t0)
	t0.Square(&t1)
	t3.Square(&t0)
	t0.Mul(&t1, &t3)
	t3.Mul(&t2, &t0)
	t0.Square(&t3)
	t2.Mul(&t1, &t0)
	t0.Square(&t2)
	t1.Mul(&t3, &t0)
	t0.Square(&t1)
	t3.Mul(&t1, &t0)
	t0.Mul(&t2, &t3)
	t2.Mul(&t1, &t0)
	t1.Square(&t2)
	t3.Square(&t1)
	t4.Square(&t3)
	t3.Mul(&t1, &t4)
	t1.Mul(&t0, &t3)
	t0.Mul(&t2, &t1)
	t2.Mul(&t1, &t0)
	t1.Square(&t2)
	t3.Square(&t1)
	t1.Mul(&t0, &t3)
	t0.Square(&t1)
	t3.Square(&t0)
	t0.Mul(&t1, &t3)
	t3.Mul(&t2, &t0)
	t0.Square(&t3)
	t2.Mul(&t1, &t0)
	t0.Square(&t2)
	t1.Square(&t0)
	t0.Mul(&t2, &t1)
	t1.Mul(&t3, &t0)
	t0.Square(&t1)
	t3.Square(&t0)
	t0.Square(&t3)
	t3.Square(&t0)
	t0.Square(&t3)
	t3.Square(&t0)
	t0.Mul(&t1, &t3)
	t3.Mul(&t2, &t0)
	t0.Square(&t3)
	t2.Mul(&t1, &t0)
	t0.Square(&t2)
	t1.Mul(&t2, &t0)
	t0.Square(&t1)
	t4.Mul(&t2, &t0)
	t0.Square(&t4)
	t4.Square(&t0)
	t0.Mul(&t1, &t4)
	t1.Mul(&t3, &t0)
	t0.Square(&t1)
	t3.Mul(&t1, &t0)
	t0.Square(&t3)
	t4.Square(&t0)
	t0.Mul(&t3, &t4)
	t3.Mul(&t2, &t0)
	t0.Square(&t3)
	t2.Square(&t0)
	t0.Square(&t2)
	t2.Mul(&t1, &t0)
	t0.Square(&t2)
	t1.Mul(&t3, &t0)
	t0.Mul(&t2, &t1)
	t2.Mul(&t1, &t0)
	t1.Square(&t2)
	t3.Square(&t1)
	t1.Mul(&t0, &t3)
	t0.Square(&t1)
	t3.Mul(&t2, &t0)
	t0.Mul(&t1, &t3)
	t1.Square(&t0)
	t2.Square(&t1)
	t1.Mul(&t0, &t2)
	t2.Mul(&t3, &t1)
	t1.Mul(&t0, &t2)
	t0.Mul(&t2, &t1)
	t2.Square(&t0)
	t3.Mul(&t0, &t2)
	t2.Square(&t3)
	t3.Mul(&t1, &t2)
	t1.Mul(&t0, &t3)
	t0.Square(&t1)
	t2.Mul(&t1, &t0)
	t0.Square(&t2)
	t4.Mul(&t2, &t0)
	t0.Square(&t4)
	t4.Square(&t0)
	t5.Square(&t4)
	t4.Square(&t5)
	t5.Square(&t4)
	t4.Square(&t5)
	t5.Mul(&t0, &t4)
	t0.Mul(&t2, &t5)
	t2.Mul(&t3, &t0)
	t0.Mul(&t1, &t2)
	t1.Square(&t0)
	t3.Mul(&t0, &t1)
	t1.Square(&t3)
	t4.Mul(&t0, &t1)
	t1.Square(&t4)
	t4.Square(&t1)
	t1.Square(&t4)
	t4.Mul(&t3, &t1)
	t1.Mul(&t2, &t4)
	t2.Square(&t1)
	t3.Square(&t2)
	t4.Square(&t3)
	t3.Mul(&t2, &t4)
	t2.Mul(&t1, &t3)
	t3.Mul(&t0, &t2)
	t0.Square(&t3)
	t2.Square(&t0)
	t0.Square(&t2)
	t2.Mul(&t1, &t0)
	t0.Mul(&t3, &t2)
	t1.Square(&t0)
	t3.Square(&t1)
	t4.Mul(&t1, &t3)
	t3.Square(&t4)
	t4.Square(&t3)
	t3.Mul(&t1, &t4)
	t1.Mul(&t2, &t3)
	t2.Square(&t1)
	t3.Square(&t2)
	t2.Mul(&t0, &t3)
	t0.Square(&t2)
	t3.Mul(&t1, &t0)
	t0.Square(&t3)
	t1.Mul(&t2, &t0)
	t0.Mul(&t3, &t1)
	t2.Square(&t0)
	t3.Square(&t2)
	t2.Square(&t3)
	t3.Square(&t2)
	t2.Mul(&t1, &t3)
	t1.Mul(&t0, &t2)
	t0.Square(&t1)
	t3.Square(&t0)
	t4.Square(&t3)
	t3.Mul(&t0, &t4)
	t0.Mul(&t1, &t3)
	t3.Mul(&t2, &t0)
	t0.Square(&t3)
	t2.Square(&t0)
	t0.Mul(&t1, &t2)
	t1.Square(&t0)
	t2.Mul(&t3, &t1)
	t1.Mul(&t0, &t2)
	t0.Square(&t1)
	t3.Mul(&t2, &t0)
	t0.Square(&t3)
	t2.Square(&t0)
	t0.Mul(&t1, &t2)
	t1.Mul(&t3, &t0)
	t2.Square(&t1)
	t3.Mul(&t0, &t2)
	t0.Mul(&t1, &t3)
	t1.Square(&t0)
	t2.Square(&t1)
	t4.Square(&t2)
	t2.Mul(&t1, &t4)
	t4.Square(&t2)
	t2.Square(&t4)
	t4.Square(&t2)
	t2.Mul(&t1, &t4)
	t1.Mul(&t3, &t2)
	t2.Square(&t1)
	t3.Square(&t2)
	t2.Mul(&t1, &t3)
	t3.Square(&t2)
	t2.Square(&t3)
	t3.Mul(&t1, &t2)
	t2.Mul(&t0, &t3)
	t0.Square(&t2)
	t3.Mul(&t2, &t0)
	t0.Square(&t3)
	t4.Square(&t0)
	t0.Mul(&t3, &t4)
	t3.Mul(&t1, &t0)
	t0.Square(&t3)
	t1.Mul(&t3, &t0)
	t0.Mul(&t2, &t1)
	for i := 0; i < 126; i++ {
		t0.Square(&t0)
	}
	s.Mul(&t3, &t0)
	return s
}

// IsNonZeroI returns 1 if s is non-zero and 0 otherwise.
func (s *Scalar) IsNonZeroI() int32 {
	var ret uint8
	ret = (s[0] | s[1] | s[2] | s[3] | s[4] | s[5] | s[6] |
		s[7] | s[8] | s[9] | s[10] | s[11] | s[12] | s[13] |
		s[14] | s[15] | s[16] | s[17] | s[18] | s[19] | s[20] |
		s[21] | s[22] | s[23] | s[24] | s[25] | s[26] | s[27] |
		s[28] | s[29] | s[30] | s[31])
	ret |= ret >> 4
	ret |= ret >> 2
	ret |= ret >> 1
	return int32(ret & 1)
}

// EqualsI returns 1 if s is equal to a, otherwise 0.
func (s *Scalar) EqualsI(a *Scalar) int32 {
	var b Scalar
	return 1 - b.Sub(s, a).IsNonZeroI()
}

// Equals returns whether s is equal to a.
func (s *Scalar) Equals(a *Scalar) bool {
	var b Scalar
	return b.Sub(s, a).IsNonZeroI() == 0
}

// Implements encoding/BinaryUnmarshaler. Use SetBytes, if convenient, instead.
func (s *Scalar) UnmarshalBinary(data []byte) error {
	if len(data) != 32 {
		return fmt.Errorf("ristretto.Scalar should be 32 bytes; not %d", len(data))
	}
	var buf [32]byte
	copy(buf[:], data)
	s.SetBytes(&buf)
	return nil
}

// Implements encoding/BinaryMarshaler. Use BytesInto, if convenient, instead.
func (s *Scalar) MarshalBinary() ([]byte, error) {
	var buf [32]byte
	s.BytesInto(&buf)
	return buf[:], nil
}

func (s *Scalar) MarshalText() ([]byte, error) {
	enc := base64.RawURLEncoding
	var buf [32]byte
	s.BytesInto(&buf)
	ret := make([]byte, enc.EncodedLen(32))
	enc.Encode(ret, buf[:])
	return ret, nil
}

func (s *Scalar) UnmarshalText(txt []byte) error {
	enc := base64.RawURLEncoding
	var buf [32]byte
	n, err := enc.Decode(buf[:], txt)
	if err != nil {
		return err
	}
	if n != 32 {
		return fmt.Errorf("ristretto.Scalar should be 32 bytes; not %d", n)
	}
	s.SetBytes(&buf)
	return nil
}

func (s Scalar) String() string {
	text, _ := s.MarshalText()
	return string(text)
}
