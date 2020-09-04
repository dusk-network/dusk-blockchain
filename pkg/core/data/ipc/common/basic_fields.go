package common

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// BlsScalar represents a number in the BLS scalar field.
type BlsScalar struct {
	Data []byte `json:"data"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (b *BlsScalar) Copy() *BlsScalar {
	data := make([]byte, len(b.Data))
	copy(data, b.Data)
	return &BlsScalar{data}
}

// MBlsScalar copies the BlsScalar structure into the Rusk equivalent.
func MBlsScalar(r *rusk.BlsScalar, b *BlsScalar) {
	data := make([]byte, len(b.Data))
	copy(data, b.Data)
	r.Data = data
}

// UBlsScalar copies the Rusk BlsScalar into the native equivalent.
func UBlsScalar(r *rusk.BlsScalar, b *BlsScalar) {
	data := make([]byte, len(r.Data))
	copy(data, r.Data)
	b.Data = data
}

// MarshalBlsScalar writes the BlsScalar struct into a bytes.Buffer.
// Because BLS scalars are always supposed to be 32 bytes, we simply
// use the `Write256` encoding function.
func MarshalBlsScalar(r *bytes.Buffer, b *BlsScalar) error {
	return encoding.Write256(r, b.Data)
}

// UnmarshalBlsScalar reads a BlsScalar struct from a bytes.Buffer.
// Because BLS scalars are always supposed to be 32 bytes, we simply
// use the `Read256` encoding function.
func UnmarshalBlsScalar(r *bytes.Buffer, b *BlsScalar) error {
	b.Data = make([]byte, 32)
	return encoding.Read256(r, b.Data)
}

// JubJubScalar represents a number in the JubJub scalar field.
type JubJubScalar struct {
	Data []byte `json:"data"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (j *JubJubScalar) Copy() *JubJubScalar {
	data := make([]byte, len(j.Data))
	copy(data, j.Data)
	return &JubJubScalar{data}
}

// MJubJubScalar copies the JubJubScalar structure into the Rusk equivalent.
func MJubJubScalar(r *rusk.JubJubScalar, b *JubJubScalar) {
	data := make([]byte, len(b.Data))
	copy(data, b.Data)
	r.Data = data
}

// UJubJubScalar copies the Rusk JubJubScalar into the native equivalent.
func UJubJubScalar(r *rusk.JubJubScalar, b *JubJubScalar) {
	data := make([]byte, len(r.Data))
	copy(data, r.Data)
	b.Data = data
}

// MarshalJubJubScalar writes the JubJubScalar struct into a bytes.Buffer.
// Because JubJub scalars are always supposed to be 32 bytes, we simply
// use the `Write256` encoding function.
func MarshalJubJubScalar(r *bytes.Buffer, b *JubJubScalar) error {
	return encoding.Write256(r, b.Data)
}

// UnmarshalJubJubScalar reads a JubJubScalar struct from a bytes.Buffer.
// Because JubJub scalars are always supposed to be 32 bytes, we simply
// use the `Read256` encoding function.
func UnmarshalJubJubScalar(r *bytes.Buffer, b *JubJubScalar) error {
	b.Data = make([]byte, 32)
	return encoding.Read256(r, b.Data)
}

// JubJubCompressed represents a compressed point on the JubJub curve.
type JubJubCompressed struct {
	Data []byte `json:"data"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (j *JubJubCompressed) Copy() *JubJubCompressed {
	data := make([]byte, len(j.Data))
	copy(data, j.Data)
	return &JubJubCompressed{data}
}

// MJubJubCompressed copies the JubJubCompressed structure into the Rusk equivalent.
func MJubJubCompressed(r *rusk.JubJubCompressed, b *JubJubCompressed) {
	data := make([]byte, len(b.Data))
	copy(data, b.Data)
	r.Data = data
}

// UJubJubCompressed copies the Rusk JubJubCompressed into the native equivalent.
func UJubJubCompressed(r *rusk.JubJubCompressed, b *JubJubCompressed) {
	data := make([]byte, len(r.Data))
	copy(data, r.Data)
	b.Data = data
}

// MarshalJubJubCompressed writes the JubJubCompressed struct into a bytes.Buffer.
// Because JubJub compressed points are always supposed to be 32 bytes, we simply
// use the `Write256` encoding function.
func MarshalJubJubCompressed(r *bytes.Buffer, b *JubJubCompressed) error {
	return encoding.Write256(r, b.Data)
}

// UnmarshalJubJubCompressed reads a JubJubCompressed struct from a bytes.Buffer.
// Because JubJub compressed points are always supposed to be 32 bytes, we simply
// use the `Read256` encoding function.
func UnmarshalJubJubCompressed(r *bytes.Buffer, b *JubJubCompressed) error {
	b.Data = make([]byte, 32)
	return encoding.Read256(r, b.Data)
}

// PoseidonCipher ...
type PoseidonCipher struct {
	Data []byte `json:"data"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (p *PoseidonCipher) Copy() *PoseidonCipher {
	data := make([]byte, len(p.Data))
	copy(data, p.Data)
	return &PoseidonCipher{data}
}

// MPoseidonCipher copies the PoseidonCipher structure into the Rusk equivalent.
func MPoseidonCipher(r *rusk.PoseidonCipher, b *PoseidonCipher) {
	data := make([]byte, len(b.Data))
	copy(data, b.Data)
	r.Data = data
}

// UPoseidonCipher copies the Rusk PoseidonCipher into the native equivalent.
func UPoseidonCipher(r *rusk.PoseidonCipher, b *PoseidonCipher) {
	data := make([]byte, len(r.Data))
	copy(data, r.Data)
	b.Data = data
}

// MarshalPoseidonCipher writes the PoseidonCipher struct into a bytes.Buffer.
func MarshalPoseidonCipher(r *bytes.Buffer, b *PoseidonCipher) error {
	return encoding.WriteVarBytes(r, b.Data)
}

// UnmarshalPoseidonCipher reads a PoseidonCipher struct from a bytes.Buffer.
func UnmarshalPoseidonCipher(r *bytes.Buffer, b *PoseidonCipher) error {
	return encoding.ReadVarBytes(r, &b.Data)
}

// Proof holds the zero-knowledge proof data, typically a PLONK proof.
type Proof struct {
	Data []byte `json:"data"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (p *Proof) Copy() *Proof {
	data := make([]byte, len(p.Data))
	copy(data, p.Data)
	return &Proof{data}
}

// MProof copies the Proof structure into the Rusk equivalent.
func MProof(r *rusk.Proof, b *Proof) {
	data := make([]byte, len(b.Data))
	copy(data, b.Data)
	r.Data = data
}

// UProof copies the Rusk Proof into the native equivalent.
func UProof(r *rusk.Proof, b *Proof) {
	data := make([]byte, len(r.Data))
	copy(data, r.Data)
	b.Data = data
}

// MarshalProof writes the Proof struct into a bytes.Buffer.
func MarshalProof(r *bytes.Buffer, b *Proof) error {
	return encoding.WriteVarBytes(r, b.Data)
}

// UnmarshalProof reads a Proof struct from a bytes.Buffer.
func UnmarshalProof(r *bytes.Buffer, b *Proof) error {
	return encoding.ReadVarBytes(r, &b.Data)
}
