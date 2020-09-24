package keys

import (
	"bytes"
	"encoding/hex"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// SecretKey is a Phoenix secret key, consisting of two JubJub scalars.
type SecretKey struct {
	A *common.JubJubScalar `json:"a"`
	B *common.JubJubScalar `json:"b"`
}

// NewSecretKey returns a new empty SecretKey struct.
func NewSecretKey() *SecretKey {
	return &SecretKey{
		A: common.NewJubJubScalar(),
		B: common.NewJubJubScalar(),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (s *SecretKey) Copy() *SecretKey {
	return &SecretKey{
		A: s.A.Copy(),
		B: s.B.Copy(),
	}
}

// MSecretKey copies the SecretKey structure into the Rusk equivalent.
func MSecretKey(r *rusk.SecretKey, f *SecretKey) {
	common.MJubJubScalar(r.A, f.A)
	common.MJubJubScalar(r.B, f.B)
}

// USecretKey copies the Rusk SecretKey structure into the native equivalent.
func USecretKey(r *rusk.SecretKey, f *SecretKey) {
	common.UJubJubScalar(r.A, f.A)
	common.UJubJubScalar(r.B, f.B)
}

// MarshalSecretKey writes the SecretKey struct into a bytes.Buffer.
func MarshalSecretKey(r *bytes.Buffer, f *SecretKey) error {
	if err := common.MarshalJubJubScalar(r, f.A); err != nil {
		return err
	}

	return common.MarshalJubJubScalar(r, f.B)
}

// UnmarshalSecretKey reads a SecretKey struct from a bytes.Buffer.
func UnmarshalSecretKey(r *bytes.Buffer, f *SecretKey) error {
	if err := common.UnmarshalJubJubScalar(r, f.A); err != nil {
		return err
	}

	return common.UnmarshalJubJubScalar(r, f.B)
}

// ViewKey is a Phoenix view key, consisting of a JubJub scalar, and a
// JubJub point.
type ViewKey struct {
	A  *common.JubJubScalar     `json:"a"`
	BG *common.JubJubCompressed `json:"b_g"`
}

// NewViewKey returns a new empty ViewKey struct.
func NewViewKey() *ViewKey {
	return &ViewKey{
		A:  common.NewJubJubScalar(),
		BG: common.NewJubJubCompressed(),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (v *ViewKey) Copy() *ViewKey {
	return &ViewKey{
		A:  v.A.Copy(),
		BG: v.BG.Copy(),
	}
}

// MViewKey copies the ViewKey structure into the Rusk equivalent.
func MViewKey(r *rusk.ViewKey, f *ViewKey) {
	common.MJubJubScalar(r.A, f.A)
	common.MJubJubCompressed(r.BG, f.BG)
}

// UViewKey copies the Rusk ViewKey structure into the native equivalent.
func UViewKey(r *rusk.ViewKey, f *ViewKey) {
	common.UJubJubScalar(r.A, f.A)
	common.UJubJubCompressed(r.BG, f.BG)
}

// MarshalViewKey writes the ViewKey struct into a bytes.Buffer.
func MarshalViewKey(r *bytes.Buffer, f *ViewKey) error {
	if err := common.MarshalJubJubScalar(r, f.A); err != nil {
		return err
	}

	return common.MarshalJubJubCompressed(r, f.BG)
}

// UnmarshalViewKey reads a ViewKey struct from a bytes.Buffer.
func UnmarshalViewKey(r *bytes.Buffer, f *ViewKey) error {
	if err := common.UnmarshalJubJubScalar(r, f.A); err != nil {
		return err
	}

	return common.UnmarshalJubJubCompressed(r, f.BG)
}

// PublicKey is a Phoenix public key, consisting of two JubJub points.
type PublicKey struct {
	AG *common.JubJubCompressed `json:"a_g"`
	BG *common.JubJubCompressed `json:"b_g"`
}

// NewPublicKey returns a new empty PublicKey struct.
func NewPublicKey() *PublicKey {
	return &PublicKey{
		AG: common.NewJubJubCompressed(),
		BG: common.NewJubJubCompressed(),
	}
}

// ToAddr concatenates the two points in the PublicKey, and returns them
// as a hex-encoded slice of bytes.
func (p PublicKey) ToAddr() []byte {
	c := append(p.AG.Data, p.BG.Data...)
	return []byte(hex.EncodeToString(c))
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (p *PublicKey) Copy() *PublicKey {
	return &PublicKey{
		AG: p.AG.Copy(),
		BG: p.BG.Copy(),
	}
}

// MPublicKey copies the PublicKey structure into the Rusk equivalent.
func MPublicKey(r *rusk.PublicKey, f *PublicKey) {
	common.MJubJubCompressed(r.AG, f.AG)
	common.MJubJubCompressed(r.BG, f.BG)
}

// UPublicKey copies the Rusk PublicKey structure into the native equivalent.
func UPublicKey(r *rusk.PublicKey, f *PublicKey) {
	common.UJubJubCompressed(r.AG, f.AG)
	common.UJubJubCompressed(r.BG, f.BG)
}

// MarshalPublicKey writes the PublicKey struct into a bytes.Buffer.
func MarshalPublicKey(r *bytes.Buffer, f *PublicKey) error {
	if err := common.MarshalJubJubCompressed(r, f.AG); err != nil {
		return err
	}

	return common.MarshalJubJubCompressed(r, f.BG)
}

// UnmarshalPublicKey reads a PublicKey struct from a bytes.Buffer.
func UnmarshalPublicKey(r *bytes.Buffer, f *PublicKey) error {
	if err := common.UnmarshalJubJubCompressed(r, f.AG); err != nil {
		return err
	}

	return common.UnmarshalJubJubCompressed(r, f.BG)
}

// StealthAddress is a Phoenix stealth address, consisting of two JubJub points.
type StealthAddress struct {
	RG  *common.JubJubCompressed `json:"r_g"`
	PkR *common.JubJubCompressed `json:"pk_r"`
}

// NewStealthAddress returns a new empty StealthAddress struct.
func NewStealthAddress() *StealthAddress {
	return &StealthAddress{
		RG:  common.NewJubJubCompressed(),
		PkR: common.NewJubJubCompressed(),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (s *StealthAddress) Copy() *StealthAddress {
	return &StealthAddress{
		RG:  s.RG.Copy(),
		PkR: s.PkR.Copy(),
	}
}

// MStealthAddress copies the StealthAddress structure into the Rusk equivalent.
func MStealthAddress(r *rusk.StealthAddress, f *StealthAddress) {
	r.RG = new(rusk.JubJubCompressed)
	common.MJubJubCompressed(r.RG, f.RG)
	r.PkR = new(rusk.JubJubCompressed)
	common.MJubJubCompressed(r.PkR, f.PkR)
}

// UStealthAddress copies the Rusk StealthAddress structure into the native equivalent.
func UStealthAddress(r *rusk.StealthAddress, f *StealthAddress) {
	common.UJubJubCompressed(r.RG, f.RG)
	common.UJubJubCompressed(r.PkR, f.PkR)
}

// MarshalStealthAddress writes the StealthAddress struct into a bytes.Buffer.
func MarshalStealthAddress(r *bytes.Buffer, f *StealthAddress) error {
	if err := common.MarshalJubJubCompressed(r, f.RG); err != nil {
		return err
	}

	return common.MarshalJubJubCompressed(r, f.PkR)
}

// UnmarshalStealthAddress reads a StealthAddress struct from a bytes.Buffer.
func UnmarshalStealthAddress(r *bytes.Buffer, f *StealthAddress) error {
	if err := common.UnmarshalJubJubCompressed(r, f.RG); err != nil {
		return err
	}

	return common.UnmarshalJubJubCompressed(r, f.PkR)
}
