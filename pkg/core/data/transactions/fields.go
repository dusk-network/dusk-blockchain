package transactions

import (
	"bytes"
	"encoding/hex"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// NoteType is either Transparent or Obfuscated
type NoteType int32

const (
	// TRANSPARENT denotes a public note (transaction output)
	TRANSPARENT NoteType = 0
	// OBFUSCATED denotes a private note (transaction output)
	OBFUSCATED NoteType = 1
)

// Note is the spendable output
type Note struct {
	NoteType                  NoteType         `json:"note_type"`
	Pos                       uint64           `json:"pos"`
	Nonce                     *Nonce           `json:"nonce"`
	RG                        *CompressedPoint `json:"r_g"`
	PkR                       *CompressedPoint `json:"pk_r"`
	ValueCommitment           *Scalar          `json:"value_commitment"`
	TransparentBlindingFactor *Scalar          `json:"transparent_blinding_factor"`
	EncryptedBlindingFactor   []byte           `json:"encrypted_blinding_factor"`
	TransparentValue          uint64           `json:"transparent_value"`
	EncryptedValue            []byte           `json:"encrypted_value"`
}

//MNote to rusk
func MNote(r *rusk.Note, n *Note) error {
	switch n.NoteType {
	case TRANSPARENT:
		r.NoteType = rusk.NoteType_TRANSPARENT
	case OBFUSCATED:
		r.NoteType = rusk.NoteType_OBFUSCATED
	default:
		return errors.New("Unexpected Note type")
	}
	r.Pos = n.Pos
	r.Nonce = new(rusk.Nonce)
	MNonce(r.Nonce, n.Nonce)
	r.RG = new(rusk.CompressedPoint)
	MCompressedPoint(r.RG, n.RG)
	r.PkR = new(rusk.CompressedPoint)
	MCompressedPoint(r.PkR, n.PkR)
	r.ValueCommitment = new(rusk.Scalar)
	MScalar(r.ValueCommitment, n.ValueCommitment)

	// If note is TRANSPARENT
	if n.NoteType == TRANSPARENT {
		tbf := new(rusk.Scalar)
		MScalar(tbf, n.TransparentBlindingFactor)
		r.BlindingFactor = &rusk.Note_TransparentBlindingFactor{
			TransparentBlindingFactor: tbf,
		}

		r.Value = &rusk.Note_TransparentValue{
			TransparentValue: n.TransparentValue,
		}
		return nil
	}

	// Otherwise if note is OBFUSCATED
	ebf := make([]byte, len(n.EncryptedBlindingFactor))
	copy(ebf, n.EncryptedBlindingFactor)
	r.BlindingFactor = &rusk.Note_EncryptedBlindingFactor{
		EncryptedBlindingFactor: ebf,
	}

	ev := make([]byte, len(n.EncryptedValue))
	copy(ev, n.EncryptedValue)
	r.Value = &rusk.Note_EncryptedValue{
		EncryptedValue: ev,
	}
	return nil
}

//UNote to a buffer
func UNote(r *rusk.Note, n *Note) error {
	switch r.NoteType {
	case rusk.NoteType_TRANSPARENT:
		n.NoteType = TRANSPARENT
	case rusk.NoteType_OBFUSCATED:
		n.NoteType = OBFUSCATED
	default:
		return errors.New("Unexpected Note type")
	}
	n.Pos = r.Pos
	n.Nonce = new(Nonce)
	UNonce(r.Nonce, n.Nonce)
	n.RG = new(CompressedPoint)
	UCompressedPoint(r.RG, n.RG)
	n.PkR = new(CompressedPoint)
	UCompressedPoint(r.PkR, n.PkR)
	n.ValueCommitment = new(Scalar)
	UScalar(r.ValueCommitment, n.ValueCommitment)

	// If note is TRANSPARENT
	if n.NoteType == TRANSPARENT {
		bf, ok := r.BlindingFactor.(*rusk.Note_TransparentBlindingFactor)
		if !ok {
			return errors.New("transparent blinding factor cannot be nil for transparent notes")
		}
		n.TransparentBlindingFactor = new(Scalar)
		UScalar(bf.TransparentBlindingFactor, n.TransparentBlindingFactor)
		v, right := r.Value.(*rusk.Note_TransparentValue)
		if !right {
			return errors.New("transparent value cannot be nil for transparent notes")
		}
		n.TransparentValue = v.TransparentValue
		return nil
	}

	// Otherwise if note is OBFUSCATED
	bf, ok := r.BlindingFactor.(*rusk.Note_EncryptedBlindingFactor)
	if !ok {
		return errors.New("encrypted blinding factor cannot be nil for obfuscated notes")
	}

	n.EncryptedBlindingFactor = make([]byte, len(bf.EncryptedBlindingFactor))
	copy(n.EncryptedBlindingFactor, bf.EncryptedBlindingFactor)

	v, right := r.Value.(*rusk.Note_EncryptedValue)
	if !right {
		return errors.New("encrypted value cannot be nil for obfuscated notes")
	}

	n.EncryptedValue = make([]byte, len(v.EncryptedValue))
	copy(n.EncryptedValue, v.EncryptedValue)
	return nil
}

//MarshalNote to a buffer
func MarshalNote(r *bytes.Buffer, n Note) error {
	var err error

	switch n.NoteType {
	case TRANSPARENT:
		err = encoding.WriteUint8(r, uint8(0))
	case OBFUSCATED:
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

	// If note is TRANSPARENT
	if n.NoteType == TRANSPARENT {
		if err := MarshalScalar(r, *n.TransparentBlindingFactor); err != nil {
			return err
		}
		if err := encoding.WriteUint64LE(r, n.TransparentValue); err != nil {
			return err
		}

		return nil
	}

	// Otherwise if note is OBFUSCATED
	if err := encoding.WriteVarBytes(r, n.EncryptedBlindingFactor); err != nil {
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
		n.NoteType = TRANSPARENT
	} else if raw == uint8(1) {
		n.NoteType = OBFUSCATED
	} else {
		return errors.New("Buffer cannot unmarshal a valid note type")
	}

	if err := encoding.ReadUint64LE(r, &n.Pos); err != nil {
		return err
	}

	n.Nonce = new(Nonce)
	if err := UnmarshalNonce(r, n.Nonce); err != nil {
		return err
	}

	n.RG = new(CompressedPoint)
	if err := UnmarshalCompressedPoint(r, n.RG); err != nil {
		return err
	}

	n.PkR = new(CompressedPoint)
	if err := UnmarshalCompressedPoint(r, n.PkR); err != nil {
		return err
	}

	n.ValueCommitment = new(Scalar)
	if err := UnmarshalScalar(r, n.ValueCommitment); err != nil {
		return err
	}

	// If note is TRANSPARENT
	if n.NoteType == TRANSPARENT {

		n.TransparentBlindingFactor = new(Scalar)
		if err := UnmarshalScalar(r, n.TransparentBlindingFactor); err != nil {
			return err
		}
		if err := encoding.ReadUint64LE(r, &n.TransparentValue); err != nil {
			return err
		}

		return nil
	}

	// Otherwise if it is OBFUSCATED
	if err := encoding.ReadVarBytes(r, &n.EncryptedBlindingFactor); err != nil {
		return err
	}
	if err := encoding.ReadVarBytes(r, &n.EncryptedValue); err != nil {
		return err
	}
	return nil
}

// Nullifier of the transaction
type Nullifier struct {
	H *Scalar `json:"h"`
}

// MNullifier copy the Nullifier from rusk to transactions datastruct
func MNullifier(r *rusk.Nullifier, n *Nullifier) {
	r.H = new(rusk.Scalar)
	MScalar(r.H, n.H)
}

// UNullifier copy the Nullifier from rusk to transactions datastruct
func UNullifier(r *rusk.Nullifier, n *Nullifier) {
	n.H = new(Scalar)
	UScalar(r.H, n.H)
}

// MarshalNullifier to a buffer
func MarshalNullifier(r *bytes.Buffer, n Nullifier) error {
	return MarshalScalar(r, *n.H)
}

// UnmarshalNullifier from a buffer
func UnmarshalNullifier(r *bytes.Buffer, n *Nullifier) error {
	n.H = new(Scalar)
	return UnmarshalScalar(r, n.H)
}

// SecretKey to sign the ContractCall
type SecretKey struct {
	A *Scalar `json:"a"`
	B *Scalar `json:"b"`
}

// IsEmpty is used when passing the SecretKey by value
func (s SecretKey) IsEmpty() bool {
	return s.A == nil && s.B == nil
}

// MSecretKey copy the SecretKey from transactions datastruct to rusk.SecretKey
func MSecretKey(r *rusk.SecretKey, n *SecretKey) {
	if n.IsEmpty() {
		return
	}
	r.A = new(rusk.Scalar)
	MScalar(r.A, n.A)
	r.B = new(rusk.Scalar)
	MScalar(r.B, n.B)
}

// USecretKey copy the SecretKey from rusk datastruct to transactions.SecretKey
func USecretKey(r *rusk.SecretKey, n *SecretKey) {
	n.A = new(Scalar)
	UScalar(r.A, n.A)
	n.B = new(Scalar)
	UScalar(r.B, n.B)
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
	n.A = new(Scalar)
	if err := UnmarshalScalar(r, n.A); err != nil {
		return err
	}
	n.B = new(Scalar)
	return UnmarshalScalar(r, n.B)
}

// ViewKey is to view the transactions belonging to the related SecretKey
type ViewKey struct {
	A  *Scalar          `json:"a"`
	BG *CompressedPoint `json:"b_g"`
}

// MViewKey copies the ViewKey from rusk to transactions datastructs
func MViewKey(r *rusk.ViewKey, n *ViewKey) {
	r.A = new(rusk.Scalar)
	MScalar(r.A, n.A)
	r.BG = new(rusk.CompressedPoint)
	MCompressedPoint(r.BG, n.BG)
}

// UViewKey copies the ViewKey from rusk to transactions datastructs
func UViewKey(r *rusk.ViewKey, n *ViewKey) {
	n.A = new(Scalar)
	UScalar(r.A, n.A)
	n.BG = new(CompressedPoint)
	UCompressedPoint(r.BG, n.BG)
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
	n.A = new(Scalar)
	if err := UnmarshalScalar(r, n.A); err != nil {
		return err
	}
	n.BG = new(CompressedPoint)
	return UnmarshalCompressedPoint(r, n.BG)
}

// PublicKey is the public key
type PublicKey struct {
	AG *CompressedPoint `json:"a_g"`
	BG *CompressedPoint `json:"b_g"`
}

// IsEmpty is used to check whether the public key is empty
func (pk PublicKey) IsEmpty() bool {
	return pk.AG == nil && pk.BG == nil
}

// ToAddr returns the HEX encoded string representation of the concatenation of
// the two compressed points identifying the PublicKey
func (pk *PublicKey) ToAddr() []byte {
	repr := make([]byte, len(pk.AG.Y)+len(pk.BG.Y))
	return []byte(hex.EncodeToString(repr))
}

// MPublicKey copies the PublicKey from rusk to transactions datastructs
func MPublicKey(r *rusk.PublicKey, n *PublicKey) {
	r.AG = new(rusk.CompressedPoint)
	r.BG = new(rusk.CompressedPoint)
	// TODO: check if protobuf does not vomit if the PublicKey is empty (which
	// is the case for all genesis contracts)
	if n.IsEmpty() {
		return
	}
	MCompressedPoint(r.AG, n.AG)
	MCompressedPoint(r.BG, n.BG)
}

// UPublicKey copies the PublicKey from rusk to transactions datastructs
func UPublicKey(r *rusk.PublicKey, n *PublicKey) {
	n.AG = new(CompressedPoint)
	UCompressedPoint(r.AG, n.AG)
	n.BG = new(CompressedPoint)
	UCompressedPoint(r.BG, n.BG)
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
	n.AG = new(CompressedPoint)
	if err := UnmarshalCompressedPoint(r, n.AG); err != nil {
		return err
	}
	n.BG = new(CompressedPoint)
	return UnmarshalCompressedPoint(r, n.BG)
}

// Scalar of the BLS12_381 curve
type Scalar struct {
	Data []byte `json:"data"`
}

// MScalar serializes Nonce from transaction to rusk
func MScalar(r *rusk.Scalar, n *Scalar) {
	r.Data = make([]byte, len(n.Data))
	copy(r.Data, n.Data)
}

// UScalar serializes Nonce from rusk to transaction
func UScalar(r *rusk.Scalar, n *Scalar) {
	n.Data = make([]byte, len(r.Data))
	copy(n.Data, r.Data)
}

// MarshalScalar to the wire encoding
func MarshalScalar(r *bytes.Buffer, n Scalar) error {
	return encoding.WriteVarBytes(r, n.Data)
}

// UnmarshalScalar from the wire encoding
func UnmarshalScalar(r *bytes.Buffer, n *Scalar) error {
	return encoding.ReadVarBytes(r, &n.Data)
}

// CompressedPoint of the BLS12_#81 curve
type CompressedPoint struct {
	Y []byte `json:"y"`
}

// MarshalCompressedPoint to the wire encoding
func MarshalCompressedPoint(r *bytes.Buffer, n CompressedPoint) error {
	return encoding.WriteVarBytes(r, n.Y)
}

// UnmarshalCompressedPoint from the wire encoding
func UnmarshalCompressedPoint(r *bytes.Buffer, n *CompressedPoint) error {
	return encoding.ReadVarBytes(r, &n.Y)
}

// UCompressedPoint serializes Nonce from rusk to transaction
func UCompressedPoint(r *rusk.CompressedPoint, n *CompressedPoint) {
	n.Y = make([]byte, len(r.Y))
	copy(n.Y, r.Y)
}

// MCompressedPoint serializes Nonce from transaction to rusk
func MCompressedPoint(r *rusk.CompressedPoint, n *CompressedPoint) {
	r.Y = make([]byte, len(n.Y))
	copy(r.Y, n.Y)
}

// Nonce is the distributed atomic increment used to prevent double spending
type Nonce struct {
	Bs []byte `json:"bs"`
}

// MarshalNonce to the wire encoding
func MarshalNonce(r *bytes.Buffer, n Nonce) error {
	return encoding.WriteVarBytes(r, n.Bs)
}

// UnmarshalNonce from the wire encoding
func UnmarshalNonce(r *bytes.Buffer, n *Nonce) error {
	return encoding.ReadVarBytes(r, &n.Bs)
}

// MNonce serializes Nonce from rusk to transaction
func MNonce(r *rusk.Nonce, n *Nonce) {
	r.Bs = make([]byte, len(n.Bs))
	copy(r.Bs, n.Bs)
}

// UNonce serializes Nonce from rusk to transaction
func UNonce(r *rusk.Nonce, n *Nonce) {
	n.Bs = make([]byte, len(r.Bs))
	copy(n.Bs, r.Bs)
}
