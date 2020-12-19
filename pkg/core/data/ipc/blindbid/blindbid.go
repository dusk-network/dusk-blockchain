// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package blindbid

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// GenerateScoreRequest is used by block generators to generate a score
// and a proof of blind bid, in order to propose a block.
type GenerateScoreRequest struct {
	K              *common.BlsScalar        `json:"k"`
	Commitment     *common.JubJubCompressed `json:"secret"`
	Seed           *common.BlsScalar        `json:"seed"`
	Round          uint32                   `json:"round"`
	Step           uint32                   `json:"step"`
	IndexStoredBid uint64                   `json:"index_stored_bid"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (g *GenerateScoreRequest) Copy() *GenerateScoreRequest {
	return &GenerateScoreRequest{
		K:              g.K.Copy(),
		Seed:           g.Seed.Copy(),
		Commitment:     g.Commitment.Copy(),
		Round:          g.Round,
		Step:           g.Step,
		IndexStoredBid: g.IndexStoredBid,
	}
}

// MGenerateScoreRequest copies the GenerateScoreRequest structure into the Rusk equivalent.
func MGenerateScoreRequest(r *rusk.GenerateScoreRequest, f *GenerateScoreRequest) {
	r.K = new(rusk.BlsScalar)
	common.MBlsScalar(r.K, f.K)
	r.Seed = new(rusk.BlsScalar)
	common.MBlsScalar(r.Seed, f.Seed)
	r.Secret = new(rusk.JubJubCompressed)
	common.MJubJubCompressed(r.Secret, f.Commitment)
	r.Round = f.Round
	r.Step = f.Step
	r.IndexStoredBid = f.IndexStoredBid
}

// UGenerateScoreRequest copies the Rusk GenerateScoreRequest structure into the native equivalent.
func UGenerateScoreRequest(r *rusk.GenerateScoreRequest, f *GenerateScoreRequest) {
	common.UBlsScalar(r.K, f.K)
	common.UBlsScalar(r.Seed, f.Seed)
	common.UJubJubCompressed(r.Secret, f.Commitment)
	f.Round = r.Round
	f.Step = r.Step
	f.IndexStoredBid = r.IndexStoredBid
}

// MarshalGenerateScoreRequest writes the GenerateScoreRequest struct into a bytes.Buffer.
func MarshalGenerateScoreRequest(r *bytes.Buffer, f *GenerateScoreRequest) error {
	if err := common.MarshalBlsScalar(r, f.K); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.Seed); err != nil {
		return err
	}

	if err := common.MarshalJubJubCompressed(r, f.Commitment); err != nil {
		return err
	}

	if err := encoding.WriteUint32LE(r, f.Round); err != nil {
		return err
	}

	if err := encoding.WriteUint32LE(r, f.Step); err != nil {
		return err
	}

	return encoding.WriteUint64LE(r, f.IndexStoredBid)
}

// UnmarshalGenerateScoreRequest reads a GenerateScoreRequest struct from a bytes.Buffer.
func UnmarshalGenerateScoreRequest(r *bytes.Buffer, f *GenerateScoreRequest) error {
	if err := common.UnmarshalBlsScalar(r, f.K); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.Seed); err != nil {
		return err
	}

	if err := common.UnmarshalJubJubCompressed(r, f.Commitment); err != nil {
		return err
	}

	if err := encoding.ReadUint32LE(r, &f.Round); err != nil {
		return err
	}

	if err := encoding.ReadUint32LE(r, &f.Step); err != nil {
		return err
	}

	return encoding.ReadUint64LE(r, &f.IndexStoredBid)
}

// GenerateScoreResponse contains the resulting proof of blind bid, score,
// and the prover identity, to be used for proposing a block.
type GenerateScoreResponse struct {
	BlindbidProof  *common.Proof     `json:"blindbid_proof"`
	Score          *common.BlsScalar `json:"score"`
	ProverIdentity *common.BlsScalar `json:"prover_identity"`
}

// NewGenerateScoreResponse returns a new empty GenerateScoreResponse struct.
func NewGenerateScoreResponse() *GenerateScoreResponse {
	return &GenerateScoreResponse{
		BlindbidProof:  common.NewProof(),
		Score:          common.NewBlsScalar(),
		ProverIdentity: common.NewBlsScalar(),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (g *GenerateScoreResponse) Copy() *GenerateScoreResponse {
	return &GenerateScoreResponse{
		BlindbidProof:  g.BlindbidProof.Copy(),
		Score:          g.Score.Copy(),
		ProverIdentity: g.ProverIdentity.Copy(),
	}
}

// MGenerateScoreResponse copies the GenerateScoreResponse structure into the Rusk equivalent.
func MGenerateScoreResponse(r *rusk.GenerateScoreResponse, f *GenerateScoreResponse) {
	common.MProof(r.BlindbidProof, f.BlindbidProof)
	common.MBlsScalar(r.Score, f.Score)
	common.MBlsScalar(r.ProverIdentity, f.ProverIdentity)
}

// UGenerateScoreResponse copies the Rusk GenerateScoreResponse structure into the native equivalent.
func UGenerateScoreResponse(r *rusk.GenerateScoreResponse, f *GenerateScoreResponse) {
	common.UProof(r.BlindbidProof, f.BlindbidProof)
	common.UBlsScalar(r.Score, f.Score)
	common.UBlsScalar(r.ProverIdentity, f.ProverIdentity)
}

// MarshalGenerateScoreResponse writes the GenerateScoreResponse struct into a bytes.Buffer.
func MarshalGenerateScoreResponse(r *bytes.Buffer, f *GenerateScoreResponse) error {
	if err := common.MarshalProof(r, f.BlindbidProof); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.Score); err != nil {
		return err
	}

	return common.MarshalBlsScalar(r, f.ProverIdentity)
}

// UnmarshalGenerateScoreResponse reads a GenerateScoreResponse struct from a bytes.Buffer.
func UnmarshalGenerateScoreResponse(r *bytes.Buffer, f *GenerateScoreResponse) error {
	if err := common.UnmarshalProof(r, f.BlindbidProof); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.Score); err != nil {
		return err
	}

	return common.UnmarshalBlsScalar(r, f.ProverIdentity)
}

// VerifyScoreRequest is used by provisioners to ensure that a given
// blind bid proof is valid.
type VerifyScoreRequest struct {
	Proof    *common.Proof     `json:"proof"`
	Score    *common.BlsScalar `json:"score"`
	Seed     *common.BlsScalar `json:"seed"`
	ProverID *common.BlsScalar `json:"prover_id"`
	Round    uint64            `json:"round"`
	Step     uint32            `json:"step"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (v *VerifyScoreRequest) Copy() *VerifyScoreRequest {
	return &VerifyScoreRequest{
		Proof:    v.Proof.Copy(),
		Score:    v.Score.Copy(),
		Seed:     v.Seed.Copy(),
		ProverID: v.ProverID.Copy(),
		Round:    v.Round,
		Step:     v.Step,
	}
}

// MVerifyScoreRequest copies the VerifyScoreRequest structure into the Rusk equivalent.
func MVerifyScoreRequest(r *rusk.VerifyScoreRequest, f *VerifyScoreRequest) {
	r.Proof = new(rusk.Proof)
	common.MProof(r.Proof, f.Proof)
	r.Score = new(rusk.BlsScalar)
	common.MBlsScalar(r.Score, f.Score)
	r.Seed = new(rusk.BlsScalar)
	common.MBlsScalar(r.Seed, f.Seed)
	r.ProverId = new(rusk.BlsScalar)
	common.MBlsScalar(r.ProverId, f.ProverID)
	r.Round = f.Round
	r.Step = f.Step
}

// UVerifyScoreRequest copies the Rusk VerifyScoreRequest structure into the native equivalent.
func UVerifyScoreRequest(r *rusk.VerifyScoreRequest, f *VerifyScoreRequest) {
	common.UProof(r.Proof, f.Proof)
	common.UBlsScalar(r.Score, f.Score)
	common.UBlsScalar(r.Seed, f.Seed)
	common.UBlsScalar(r.ProverId, f.ProverID)
	f.Round = r.Round
	f.Step = r.Step
}

// MarshalVerifyScoreRequest writes the VerifyScoreRequest struct into a bytes.Buffer.
func MarshalVerifyScoreRequest(r *bytes.Buffer, f *VerifyScoreRequest) error {
	if err := common.MarshalProof(r, f.Proof); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.Score); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.Seed); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.ProverID); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.Round); err != nil {
		return err
	}

	return encoding.WriteUint32LE(r, f.Step)
}

// UnmarshalVerifyScoreRequest reads a VerifyScoreRequest struct from a bytes.Buffer.
func UnmarshalVerifyScoreRequest(r *bytes.Buffer, f *VerifyScoreRequest) error {
	if err := common.UnmarshalProof(r, f.Proof); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.Score); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.Seed); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.ProverID); err != nil {
		return err
	}

	if err := encoding.ReadUint64LE(r, &f.Round); err != nil {
		return err
	}

	return encoding.ReadUint32LE(r, &f.Step)
}

// VerifyScoreResponse tells a provisioner whether a supplied proof is
// valid or not.
type VerifyScoreResponse struct {
	Success bool `json:"success"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (v *VerifyScoreResponse) Copy() *VerifyScoreResponse {
	return &VerifyScoreResponse{
		Success: v.Success,
	}
}

// MVerifyScoreResponse copies the VerifyScoreResponse structure into the Rusk equivalent.
func MVerifyScoreResponse(r *rusk.VerifyScoreResponse, f *VerifyScoreResponse) {
	r.Success = f.Success
}

// UVerifyScoreResponse copies the Rusk VerifyScoreResponse structure into the native equivalent.
func UVerifyScoreResponse(r *rusk.VerifyScoreResponse, f *VerifyScoreResponse) {
	f.Success = r.Success
}

// MarshalVerifyScoreResponse writes the VerifyScoreResponse struct into a bytes.Buffer.
func MarshalVerifyScoreResponse(r *bytes.Buffer, f *VerifyScoreResponse) error {
	return encoding.WriteBool(r, f.Success)
}

// UnmarshalVerifyScoreResponse reads a VerifyScoreResponse struct from a bytes.Buffer.
func UnmarshalVerifyScoreResponse(r *bytes.Buffer, f *VerifyScoreResponse) error {
	return encoding.ReadBool(r, &f.Success)
}
