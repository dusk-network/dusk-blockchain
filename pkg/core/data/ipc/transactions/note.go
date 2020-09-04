package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Note represents a Phoenix note.
type Note struct {
	Randomness    *common.JubJubCompressed `json:"randomness"`
	PkR           *common.JubJubCompressed `json:"pk_r"`
	Commitment    *common.JubJubCompressed `json:"commitment"`
	Nonce         *common.BlsScalar        `json:"nonce"`
	EncryptedData *common.PoseidonCipher   `json:"encrypted_data"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (n *Note) Copy() *Note {
	return &Note{
		Randomness:    n.Randomness.Copy(),
		PkR:           n.PkR.Copy(),
		Commitment:    n.Commitment.Copy(),
		Nonce:         n.Nonce.Copy(),
		EncryptedData: n.EncryptedData.Copy(),
	}
}

// MNote copies the Note structure into the Rusk equivalent.
func MNote(r *rusk.Note, f *Note) {
	common.MJubJubCompressed(r.Randomness, f.Randomness)
	common.MJubJubCompressed(r.PkR, f.PkR)
	common.MJubJubCompressed(r.Commitment, f.Commitment)
	common.MBlsScalar(r.Nonce, f.Nonce)
	// XXX: fix typo in rusk-schema
	common.MPoseidonCipher(r.EncyptedData, f.EncryptedData)
}

// UNote copies the Rusk Note structure into the native equivalent.
func UNote(r *rusk.Note, f *Note) {
	common.UJubJubCompressed(r.Randomness, f.Randomness)
	common.UJubJubCompressed(r.PkR, f.PkR)
	common.UJubJubCompressed(r.Commitment, f.Commitment)
	common.UBlsScalar(r.Nonce, f.Nonce)
	// XXX: fix typo in rusk-schema
	common.UPoseidonCipher(r.EncyptedData, f.EncryptedData)
}

// MarshalNote writes the Note struct into a bytes.Buffer.
func MarshalNote(r *bytes.Buffer, f *Note) error {
	if err := common.MarshalJubJubCompressed(r, f.Randomness); err != nil {
		return err
	}

	if err := common.MarshalJubJubCompressed(r, f.PkR); err != nil {
		return err
	}

	if err := common.MarshalJubJubCompressed(r, f.Commitment); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.Nonce); err != nil {
		return err
	}

	return common.MarshalPoseidonCipher(r, f.EncryptedData)
}

// UnmarshalNote reads a Note struct from a bytes.Buffer.
func UnmarshalNote(r *bytes.Buffer, f *Note) error {
	f = new(Note)

	if err := common.UnmarshalJubJubCompressed(r, f.Randomness); err != nil {
		return err
	}

	if err := common.UnmarshalJubJubCompressed(r, f.PkR); err != nil {
		return err
	}

	if err := common.UnmarshalJubJubCompressed(r, f.Commitment); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.Nonce); err != nil {
		return err
	}

	return common.UnmarshalPoseidonCipher(r, f.EncryptedData)
}
