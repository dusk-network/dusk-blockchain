package transactions

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

// NoteType is either Transparent or Obfuscated
type NoteType int32

const (
	// NoteType_TRANSPARENT denotes a public note (transaction output)
	NoteType_TRANSPARENT NoteType = 0
	// NoteType_OBFUSCATED denotes a private note (transaction output)
	NoteType_OBFUSCATED NoteType = 1
)

// Note is the spendable output
type Note struct {
	NoteType                  NoteType         `protobuf:"varint,1,opt,name=note_type,json=noteType,proto3,enum=rusk.NoteType" json:"note_type,omitempty"`
	Pos                       uint64           `protobuf:"fixed64,2,opt,name=pos,proto3" json:"pos,omitempty"`
	Nonce                     *Nonce           `protobuf:"bytes,3,opt,name=nonce,proto3" json:"nonce,omitempty"`
	RG                        *CompressedPoint `protobuf:"bytes,4,opt,name=r_g,json=rG,proto3" json:"r_g,omitempty"`
	PkR                       *CompressedPoint `protobuf:"bytes,5,opt,name=pk_r,json=pkR,proto3" json:"pk_r,omitempty"`
	ValueCommitment           *Scalar          `protobuf:"bytes,6,opt,name=value_commitment,json=valueCommitment,proto3" json:"value_commitment,omitempty"`
	TransparentBlindingFactor *Scalar          `protobuf:"bytes,7,opt,name=transparent_blinding_factor,json=transparentBlindingFactor,proto3,oneof"`
	EncryptedBlindingFactor   []byte           `protobuf:"bytes,8,opt,name=encrypted_blinding_factor,json=encryptedBlindingFactor,proto3,oneof"`
	TransparentValue          uint64           `protobuf:"fixed64,9,opt,name=transparent_value,json=transparentValue,proto3,oneof"`
	EncryptedValue            []byte           `protobuf:"bytes,10,opt,name=encrypted_value,json=encryptedValue,proto3,oneof"`
}

//MarshalNote to a buffer
func MarshalNote(r *bytes.Buffer, n Note) error {
	var err error

	switch n.NoteType {
	case NoteType_TRANSPARENT:
		err = encoding.WriteUint8(r, uint8(0))
	case NoteType_OBFUSCATED:
		err = encoding.WriteUint8(r, uint8(1))
	default:
		return errors.New("Unexpected Note type")
	}
	if err != nil {
		return err
	}
	if err := encoding.WriteUint64LE(r, n.Pos); err != nil {
		return err
	}
	if err := MarshalNonce(r, *n.Nonce); err != nil {
		return err
	}
	if err := MarshalCompressedPoint(r, *n.RG); err != nil {
		return err
	}
	if err := MarshalCompressedPoint(r, *n.PkR); err != nil {
		return err
	}
	if err := MarshalScalar(r, *n.ValueCommitment); err != nil {
		return err
	}
	if err := MarshalScalar(r, *n.TransparentBlindingFactor); err != nil {
		return err
	}
	if err := encoding.WriteVarBytes(r, n.EncryptedBlindingFactor); err != nil {
		return err
	}
	if err := encoding.WriteUint64LE(r, n.TransparentValue); err != nil {
		return err
	}
	if err := encoding.WriteVarBytes(r, n.EncryptedValue); err != nil {
		return err
	}
	return nil
}

// UnmarshalNote from a buffer
func UnmarshalNote(r *bytes.Buffer, n *Note) error {
	var raw uint8
	if err := encoding.ReadUint8(r, &raw); err != nil {
		return err
	}

	if raw == uint8(0) {
		n.NoteType = NoteType_TRANSPARENT
	} else if raw == uint8(1) {
		n.NoteType = NoteType_OBFUSCATED
	} else {
		return errors.New("Buffer cannot unmarshal a valid note type")
	}

	if err := encoding.ReadUint64LE(r, &n.Pos); err != nil {
		return err
	}

	n.Nonce = &Nonce{}
	if err := UnmarshalNonce(r, n.Nonce); err != nil {
		return err
	}

	n.RG = &CompressedPoint{}
	if err := UnmarshalCompressedPoint(r, n.RG); err != nil {
		return err
	}

	n.PkR = &CompressedPoint{}
	if err := UnmarshalCompressedPoint(r, n.PkR); err != nil {
		return err
	}

	n.ValueCommitment = &Scalar{}
	if err := UnmarshalScalar(r, n.ValueCommitment); err != nil {
		return err
	}
	n.TransparentBlindingFactor = &Scalar{}
	if err := UnmarshalScalar(r, n.TransparentBlindingFactor); err != nil {
		return err
	}
	if err := encoding.ReadVarBytes(r, &n.EncryptedBlindingFactor); err != nil {
		return err
	}
	if err := encoding.ReadUint64LE(r, &n.TransparentValue); err != nil {
		return err
	}
	if err := encoding.ReadVarBytes(r, &n.EncryptedValue); err != nil {
		return err
	}
	return nil
}

// Nullifier of the transaction
type Nullifier struct {
	H *Scalar `protobuf:"bytes,1,opt,name=h,proto3" json:"h,omitempty"`
}

// MarshalNullifier to a buffer
func MarshalNullifier(r *bytes.Buffer, n Nullifier) error {
	return MarshalScalar(r, *n.H)
}

// UnmarshalNullifier from a buffer
func UnmarshalNullifier(r *bytes.Buffer, n *Nullifier) error {
	return UnmarshalScalar(r, n.H)
}

// SecretKey to sign the ContractCall
type SecretKey struct {
	A *Scalar `protobuf:"bytes,1,opt,name=a,proto3" json:"a,omitempty"`
	B *Scalar `protobuf:"bytes,2,opt,name=b,proto3" json:"b,omitempty"`
}

// MarshalSecretKey to the wire encoding
func MarshalSecretKey(r *bytes.Buffer, n SecretKey) error {
	if err := MarshalScalar(r, *n.A); err != nil {
		return err
	}
	return MarshalScalar(r, *n.B)
}

// UnmarshalSecretKey from the wire encoding
func UnmarshalSecretKey(r *bytes.Buffer, n *SecretKey) error {
	if err := UnmarshalScalar(r, n.A); err != nil {
		return err
	}
	return UnmarshalScalar(r, n.B)
}

// ViewKey is to view the transactions belonging to the related SecretKey
type ViewKey struct {
	A  *Scalar          `protobuf:"bytes,1,opt,name=a,proto3" json:"a,omitempty"`
	BG *CompressedPoint `protobuf:"bytes,2,opt,name=b_g,json=bG,proto3" json:"b_g,omitempty"`
}

// MarshalViewKey to the wire encoding
func MarshalViewKey(r *bytes.Buffer, n ViewKey) error {
	if err := MarshalScalar(r, *n.A); err != nil {
		return err
	}
	return MarshalCompressedPoint(r, *n.BG)
}

// UnmarshalViewKey from the wire encoding
func UnmarshalViewKey(r *bytes.Buffer, n *ViewKey) error {
	if err := UnmarshalScalar(r, n.A); err != nil {
		return err
	}
	return UnmarshalCompressedPoint(r, n.BG)
}

// PublicKey is the public key
type PublicKey struct {
	AG *CompressedPoint `protobuf:"bytes,1,opt,name=a_g,json=aG,proto3" json:"a_g,omitempty"`
	BG *CompressedPoint `protobuf:"bytes,2,opt,name=b_g,json=bG,proto3" json:"b_g,omitempty"`
}

// MarshalPublicKey to the wire encoding
func MarshalPublicKey(r *bytes.Buffer, n PublicKey) error {
	if err := MarshalCompressedPoint(r, *n.AG); err != nil {
		return err
	}
	return MarshalCompressedPoint(r, *n.BG)
}

// UnmarshalPublicKey from the wire encoding
func UnmarshalPublicKey(r *bytes.Buffer, n *PublicKey) error {
	if err := UnmarshalCompressedPoint(r, n.AG); err != nil {
		return err
	}
	return UnmarshalCompressedPoint(r, n.BG)
}

// Scalar of the BLS12_381 curve
type Scalar struct {
	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

// MarshalScalar to the wire encoding
func MarshalScalar(r *bytes.Buffer, n Scalar) error {
	return encoding.ReadVarBytes(r, &n.Data)
}

// UnmarshalScalar from the wire encoding
func UnmarshalScalar(r *bytes.Buffer, n *Scalar) error {
	return encoding.WriteVarBytes(r, n.Data)
}

// CompressedPoint of the BLS12_#81 curve
type CompressedPoint struct {
	Y []byte `protobuf:"bytes,1,opt,name=y,proto3" json:"y,omitempty"`
}

// MarshalCompressedPoint to the wire encoding
func MarshalCompressedPoint(r *bytes.Buffer, n CompressedPoint) error {
	return encoding.ReadVarBytes(r, &n.Y)
}

// UnmarshalCompressedPoint from the wire encoding
func UnmarshalCompressedPoint(r *bytes.Buffer, n *CompressedPoint) error {
	return encoding.WriteVarBytes(r, n.Y)
}

// Nonce is the distributed atomic increment used to prevent double spending
type Nonce struct {
	Bs []byte `protobuf:"bytes,1,opt,name=bs,proto3" json:"bs,omitempty"`
}

// MarshalNonce to the wire encoding
func MarshalNonce(r *bytes.Buffer, n Nonce) error {
	return encoding.ReadVarBytes(r, &n.Bs)
}

// UnmarshalNonce from the wire encoding
func UnmarshalNonce(r *bytes.Buffer, n *Nonce) error {
	return encoding.WriteVarBytes(r, n.Bs)
}
