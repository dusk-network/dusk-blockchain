// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

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

// NewNote returns a new empty Note struct.
func NewNote() *Note {
	return &Note{
		Randomness:    common.NewJubJubCompressed(),
		PkR:           common.NewJubJubCompressed(),
		Commitment:    common.NewJubJubCompressed(),
		Nonce:         common.NewBlsScalar(),
		EncryptedData: common.NewPoseidonCipher(),
	}
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
	r.Randomness = new(rusk.JubJubCompressed)
	common.MJubJubCompressed(r.Randomness, f.Randomness)
	r.PkR = new(rusk.JubJubCompressed)
	common.MJubJubCompressed(r.PkR, f.PkR)
	r.Commitment = new(rusk.JubJubCompressed)
	common.MJubJubCompressed(r.Commitment, f.Commitment)
	r.Nonce = new(rusk.BlsScalar)
	common.MBlsScalar(r.Nonce, f.Nonce)
	// XXX: fix typo in rusk-schema
	r.EncyptedData = new(rusk.PoseidonCipher)
	common.MPoseidonCipher(r.EncyptedData, f.EncryptedData)
}

// UNote copies the Rusk Note structure into the native equivalent.
func UNote(r *rusk.Note, f *Note) {
	f.Randomness = new(common.JubJubCompressed)
	common.UJubJubCompressed(r.Randomness, f.Randomness)
	f.PkR = new(common.JubJubCompressed)
	common.UJubJubCompressed(r.PkR, f.PkR)
	f.Commitment = new(common.JubJubCompressed)
	common.UJubJubCompressed(r.Commitment, f.Commitment)
	f.Nonce = new(common.BlsScalar)
	common.UBlsScalar(r.Nonce, f.Nonce)
	// XXX: fix typo in rusk-schema
	f.EncryptedData = new(common.PoseidonCipher)
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

// Equal returns whether or not two Notes are equal.
func (n *Note) Equal(other *Note) bool {
	if !n.Randomness.Equal(other.Randomness) {
		return false
	}

	if !n.PkR.Equal(other.PkR) {
		return false
	}

	if !n.Commitment.Equal(other.Commitment) {
		return false
	}

	if !n.Nonce.Equal(other.Nonce) {
		return false
	}

	return n.EncryptedData.Equal(other.EncryptedData)
}
