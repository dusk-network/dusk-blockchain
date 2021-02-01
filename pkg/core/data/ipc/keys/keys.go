// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package keys

import (
	"bytes"
	"encoding/hex"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// SecretKey is a Phoenix secret key, consisting of two JubJub scalars.
type SecretKey struct {
	A []byte `json:"a"`
	B []byte `json:"b"`
}

// NewSecretKey returns a new empty SecretKey struct.
func NewSecretKey() *SecretKey {
	return &SecretKey{
		A: make([]byte, 32),
		B: make([]byte, 32),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (s *SecretKey) Copy() *SecretKey {
	a := make([]byte, len(s.A))
	b := make([]byte, len(s.B))

	copy(a, s.A)
	copy(b, s.B)

	return &SecretKey{
		A: a,
		B: b,
	}
}

// MSecretKey copies the SecretKey structure into the Rusk equivalent.
func MSecretKey(r *rusk.SecretKey, f *SecretKey) {
	copy(r.A, f.A)
	copy(r.B, f.B)
}

// USecretKey copies the Rusk SecretKey structure into the native equivalent.
func USecretKey(r *rusk.SecretKey, f *SecretKey) {
	copy(f.A, r.A)
	copy(f.B, r.B)
}

// MarshalSecretKey writes the SecretKey struct into a bytes.Buffer.
func MarshalSecretKey(r *bytes.Buffer, f *SecretKey) error {
	if err := encoding.Write256(r, f.A); err != nil {
		return err
	}

	return encoding.Write256(r, f.B)
}

// UnmarshalSecretKey reads a SecretKey struct from a bytes.Buffer.
func UnmarshalSecretKey(r *bytes.Buffer, f *SecretKey) error {
	if err := encoding.Read256(r, f.A); err != nil {
		return err
	}

	return encoding.Read256(r, f.B)
}

// ViewKey is a Phoenix view key, consisting of a JubJub scalar, and a
// JubJub point.
type ViewKey struct {
	A  []byte `json:"a"`
	BG []byte `json:"b_g"`
}

// NewViewKey returns a new empty ViewKey struct.
func NewViewKey() *ViewKey {
	return &ViewKey{
		A:  make([]byte, 32),
		BG: make([]byte, 32),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (v *ViewKey) Copy() *ViewKey {
	a := make([]byte, len(v.A))
	bg := make([]byte, len(v.BG))

	copy(a, v.A)
	copy(bg, v.BG)

	return &ViewKey{
		A:  a,
		BG: bg,
	}
}

// MViewKey copies the ViewKey structure into the Rusk equivalent.
func MViewKey(r *rusk.ViewKey, f *ViewKey) {
	copy(r.A, f.A)
	copy(r.BG, f.BG)
}

// UViewKey copies the Rusk ViewKey structure into the native equivalent.
func UViewKey(r *rusk.ViewKey, f *ViewKey) {
	copy(f.A, r.A)
	copy(f.BG, r.BG)
}

// MarshalViewKey writes the ViewKey struct into a bytes.Buffer.
func MarshalViewKey(r *bytes.Buffer, f *ViewKey) error {
	if err := encoding.Write256(r, f.A); err != nil {
		return err
	}

	return encoding.Write256(r, f.BG)
}

// UnmarshalViewKey reads a ViewKey struct from a bytes.Buffer.
func UnmarshalViewKey(r *bytes.Buffer, f *ViewKey) error {
	if err := encoding.Read256(r, f.A); err != nil {
		return err
	}

	return encoding.Read256(r, f.BG)
}

// PublicKey is a Phoenix public key, consisting of two JubJub points.
type PublicKey struct {
	AG []byte `json:"a_g"`
	BG []byte `json:"b_g"`
}

// NewPublicKey returns a new empty PublicKey struct.
func NewPublicKey() *PublicKey {
	return &PublicKey{
		AG: make([]byte, 32),
		BG: make([]byte, 32),
	}
}

// ToAddr concatenates the two points in the PublicKey, and returns them
// as a hex-encoded slice of bytes.
func (p PublicKey) ToAddr() []byte {
	c := append(p.AG, p.BG...)
	return []byte(hex.EncodeToString(c))
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (p *PublicKey) Copy() *PublicKey {
	ag := make([]byte, len(p.AG))
	bg := make([]byte, len(p.BG))

	copy(ag, p.AG)
	copy(bg, p.BG)

	return &PublicKey{
		AG: ag,
		BG: bg,
	}
}

// MPublicKey copies the PublicKey structure into the Rusk equivalent.
func MPublicKey(r *rusk.PublicKey, f *PublicKey) {
	copy(r.AG, f.AG)
	copy(r.BG, f.BG)
}

// UPublicKey copies the Rusk PublicKey structure into the native equivalent.
func UPublicKey(r *rusk.PublicKey, f *PublicKey) {
	copy(f.AG, r.AG)
	copy(f.BG, r.BG)
}

// MarshalPublicKey writes the PublicKey struct into a bytes.Buffer.
func MarshalPublicKey(r *bytes.Buffer, f *PublicKey) error {
	if err := encoding.Write256(r, f.AG); err != nil {
		return err
	}

	return encoding.Write256(r, f.BG)
}

// UnmarshalPublicKey reads a PublicKey struct from a bytes.Buffer.
func UnmarshalPublicKey(r *bytes.Buffer, f *PublicKey) error {
	if err := encoding.Read256(r, f.AG); err != nil {
		return err
	}

	return encoding.Read256(r, f.BG)
}

// StealthAddress is a Phoenix stealth address, consisting of two JubJub points.
type StealthAddress struct {
	RG  []byte `json:"r_g"`
	PkR []byte `json:"pk_r"`
}

// NewStealthAddress returns a new empty StealthAddress struct.
func NewStealthAddress() *StealthAddress {
	return &StealthAddress{
		RG:  make([]byte, 32),
		PkR: make([]byte, 32),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (s *StealthAddress) Copy() *StealthAddress {
	rg := make([]byte, len(s.RG))
	pkR := make([]byte, len(s.PkR))

	copy(rg, s.RG)
	copy(pkR, s.PkR)

	return &StealthAddress{
		RG:  rg,
		PkR: pkR,
	}
}

// MStealthAddress copies the StealthAddress structure into the Rusk equivalent.
func MStealthAddress(r *rusk.StealthAddress, f *StealthAddress) {
	copy(r.RG, f.RG)
	copy(r.PkR, f.PkR)
}

// UStealthAddress copies the Rusk StealthAddress structure into the native equivalent.
func UStealthAddress(r *rusk.StealthAddress, f *StealthAddress) {
	copy(f.RG, r.RG)
	copy(f.PkR, r.PkR)
}

// MarshalStealthAddress writes the StealthAddress struct into a bytes.Buffer.
func MarshalStealthAddress(r *bytes.Buffer, f *StealthAddress) error {
	if err := encoding.Write256(r, f.RG); err != nil {
		return err
	}

	return encoding.Write256(r, f.PkR)
}

// UnmarshalStealthAddress reads a StealthAddress struct from a bytes.Buffer.
func UnmarshalStealthAddress(r *bytes.Buffer, f *StealthAddress) error {
	if err := encoding.Read256(r, f.RG); err != nil {
		return err
	}

	return encoding.Read256(r, f.PkR)
}
