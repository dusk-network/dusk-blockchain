// Pure Go implementation of the Ristretto prime-order group built from
// the Edwards curve Edwards25519.
//
// Many cryptographic schemes need a group of prime order.  Popular and
// efficient elliptic curves like (Edwards25519 of `ed25519` fame) are
// rarely of prime order.  There is, however, a convenient method
// to construct a prime order group from such curves, using a method
// called Ristretto proposed by Mike Hamburg.
//
// This package implements the Ristretto group constructed from Edwards25519.
// The Point type represents a group element.  The API mimics that of the
// math/big package.  For instance, to set c to a+b, one writes
//
//     var c ristretto.Point
//     c.Add(&a, &b) // sets c to a + b
//
// Most methods return the receiver, so that function can be chained:
//
//     s.Add(&a, &b).Add(&s, &c)  // sets s to a + b + c
//
// The order of the Ristretto group is l =
// 2^252 + 27742317777372353535851937790883648493 =
// 7237005577332262213973186563042994240857116359379907606001950938285454250989.
// The Scalar type implement the numbers modulo l and also has an API similar
// to math/big.

// Adapted and modified from go-ristretto
// TODO: Remove vartime scalar operations when finished

package ristretto

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/toghrulmaharramov/dusk-go/ristretto/edwards25519"
)

// Represents an element of the Ristretto group over Edwards25519.
type Point edwards25519.ExtendedPoint

// Sets p to zero (the neutral element).  Returns p.
func (p *Point) SetZero() *Point {
	p.e().SetZero()
	return p
}

// Sets p to the Edwards25519 basepoint.  Returns p
func (p *Point) SetBase() *Point {
	p.e().SetBase()
	return p
}

// Sets p to q.  Returns p
func (p *Point) Set(q *Point) *Point {
	p.e().Set(q.e())
	return p
}

// Sets p to q + r.  Returns p.
func (p *Point) Add(q, r *Point) *Point {
	p.e().Add(q.e(), r.e())
	return p
}

// Sets p to q - r.  Returns p.
func (p *Point) Sub(q, r *Point) *Point {
	p.e().Sub(q.e(), r.e())
	return p
}

// Sets p to -q.  Returns p.
func (p *Point) Neg(q *Point) *Point {
	p.e().Neg(q.e())
	return p
}

// Packs p into the given buffer.  Returns p.
func (p *Point) BytesInto(buf *[32]byte) *Point {
	p.e().RistrettoInto(buf)
	return p
}

// Returns a packed version of p.
func (p *Point) Bytes() []byte {
	return p.e().Ristretto()
}

// Sets p to the point encoded in buf using Bytes().
// Not every input encodes a point.  Returns whether the buffer encoded a point.
func (p *Point) SetBytes(buf *[32]byte) bool {
	return p.e().SetRistretto(buf)
}

// Sets p to the point corresponding to buf using the Elligator2 encoding.
//
// In contrast to SetBytes() (1) Every input buffer will decode to a point
// and (2) SetElligator() is not injective: for every point there are
// approximately four buffers that will encode to it.
func (p *Point) SetElligator(buf *[32]byte) *Point {
	var fe edwards25519.FieldElement
	var cp edwards25519.CompletedPoint
	fe.SetBytes(buf)
	cp.SetRistrettoElligator2(&fe)
	p.e().SetCompleted(&cp)
	return p
}

// Sets p to s * q.  Returns p.
func (p *Point) ScalarMult(q *Point, s *Scalar) *Point {
	p.e().ScalarMult(q.e(), (*[32]uint8)(s))
	return p
}

// Sets p to s * B, where B is the edwards25519 basepoint. Returns p.
func (p *Point) ScalarMultBase(s *Scalar) *Point {
	// TODO optimize
	var B Point
	B.SetBase()
	return p.ScalarMult(&B, s)
}

// Sets p to a random point.  Returns p.
func (p *Point) Rand() *Point {
	var buf [32]byte
	rand.Read(buf[:])
	return p.SetElligator(&buf)
}

// Sets p to the point derived from the buffer using SHA512 and Elligator2.
// Returns p.
//
// NOTE curve25519-dalek uses a different (more conservative) method to derive
// a point from raw data with a hash.  This is implemented in
// Point.DeriveDalek().
func (p *Point) Derive(buf []byte) *Point {
	var ptBuf [32]byte
	h := sha512.Sum512(buf)
	copy(ptBuf[:], h[:32])
	return p.SetElligator(&ptBuf)
}

// Sets p to the point derived from the buffer using SHA512 and Elligator2
// in the fashion of curve25519-dalek.
//
// NOTE See also Derive(), which is a different method which is twice as fast,
// but which might not be as secure as this method.
func (p *Point) DeriveDalek(data []byte) *Point {
	hash := sha512.Sum512(data)
	var p2 Point
	var buf [32]byte
	copy(buf[:], hash[:32])
	p.SetElligator(&buf)
	copy(buf[:], hash[32:])
	p2.SetElligator(&buf)
	p.Add(p, &p2)
	return p
}

// Returns 1 if p == q and 0 otherwise.
func (p *Point) EqualsI(q *Point) int32 {
	return p.e().RistrettoEqualsI(q.e())
}

// Returns whether p == q
func (p *Point) Equals(q *Point) bool {
	return p.EqualsI(q) == 1
}

// Implements encoding/BinaryUnmarshaler. Use SetBytes, if convenient, instead.
func (p *Point) UnmarshalBinary(data []byte) error {
	if len(data) != 32 {
		return fmt.Errorf("ristretto.Point should be 32 bytes; not %d", len(data))
	}
	var buf [32]byte
	copy(buf[:], data)
	if !p.SetBytes(&buf) {
		return errors.New("Buffer does not encode a ristretto.Point")
	}
	return nil
}

// Implements encoding/BinaryMarshaler. Use BytesInto, if convenient, instead.
func (p *Point) MarshalBinary() ([]byte, error) {
	var buf [32]byte
	p.BytesInto(&buf)
	return buf[:], nil
}

func (p *Point) MarshalText() ([]byte, error) {
	enc := base64.RawURLEncoding
	var buf [32]byte
	p.BytesInto(&buf)
	ret := make([]byte, enc.EncodedLen(32))
	enc.Encode(ret, buf[:])
	return ret, nil
}

func (p *Point) UnmarshalText(txt []byte) error {
	enc := base64.RawURLEncoding
	var buf [32]byte
	n, err := enc.Decode(buf[:], txt)
	if err != nil {
		return err
	}
	if n != 32 {
		return fmt.Errorf("ristretto.Point should be 32 bytes; not %d", n)
	}
	if !p.SetBytes(&buf) {
		return errors.New("Buffer does not encode a ristretto.Point")
	}
	return nil
}

func (p Point) String() string {
	text, _ := p.MarshalText()
	return string(text)
}

func (p *Point) e() *edwards25519.ExtendedPoint {
	return (*edwards25519.ExtendedPoint)(p)
}

// Identity is the identity Point
var Identity = [32]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

func (p *Point) SetIdentity() *Point {
	p.SetBytes(&Identity)
	return p
}
