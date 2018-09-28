// A C implementation of the Ristretto group based on the PandA library
// by Chuengsatiansup, Ribarski and Schwabe, which we use as a reference
// of our pure Go implementation.
// See also https://link.springer.com/chapter/10.1007/978-3-319-04873-4_14
package cref

// #include "cref.h"
import "C"

type Fe25519 C.fe25519
type GroupGe C.group_ge
type GroupScalar C.group_scalar

func (f *Fe25519) c() *C.fe25519 {
	return (*C.fe25519)(f)
}

func (f *Fe25519) Unpack(buf *[32]byte) {
	C.fe25519_unpack(f.c(), (*C.uchar)(&buf[0]))
}

func (f *Fe25519) Pack(buf *[32]byte) {
	C.fe25519_pack((*C.uchar)(&buf[0]), f.c())
}

func (g *GroupGe) c() *C.group_ge {
	return (*C.group_ge)(g)
}

func (g *GroupGe) Pack(buf *[32]byte) {
	C.group_ge_pack((*C.uchar)(&buf[0]), g.c())
}

func (g *GroupGe) Unpack(buf *[32]byte) int {
	return int(C.group_ge_unpack(g.c(), (*C.uchar)(&buf[0])))
}

func (g *GroupGe) Elligator(r0 *Fe25519) {
	C.group_ge_elligator(g.c(), r0.c())
}

func (g *GroupGe) Neg(x *GroupGe) {
	C.group_ge_negate(g.c(), x.c())
}

func (g *GroupGe) Add(x, y *GroupGe) {
	C.group_ge_add(g.c(), x.c(), y.c())
}

func (g *GroupGe) Double(x *GroupGe) {
	C.group_ge_double(g.c(), x.c())
}

func (g *GroupGe) ScalarMult(x *GroupGe, s *GroupScalar) {
	C.group_ge_scalarmult(g.c(), x.c(), s.c())
}

func (g *GroupGe) X() *Fe25519 {
	return (*Fe25519)(&g.c().x)
}

func (g *GroupGe) Y() *Fe25519 {
	return (*Fe25519)(&g.c().y)
}

func (g *GroupGe) Z() *Fe25519 {
	return (*Fe25519)(&g.c().z)
}

func (g *GroupGe) T() *Fe25519 {
	return (*Fe25519)(&g.c().t)
}

func (s *GroupScalar) c() *C.group_scalar {
	return (*C.group_scalar)(s)
}

func (s *GroupScalar) Unpack(buf *[32]byte) {
	C.group_scalar_unpack(s.c(), (*C.uchar)(&buf[0]))
}

func (s *GroupScalar) Pack(buf *[32]byte) {
	C.group_scalar_pack((*C.uchar)(&buf[0]), s.c())
}
