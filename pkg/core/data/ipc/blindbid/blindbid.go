// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package blindbid

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// GenerateScoreRequest is used by block generators to generate a score
// and a proof of blind bid, in order to propose a block.
type GenerateScoreRequest struct {
	K              []byte `json:"k"`
	Seed           []byte `json:"seed"`
	Secret         []byte `json:"secret"`
	Round          uint32 `json:"round"`
	Step           uint32 `json:"step"`
	IndexStoredBid uint64 `json:"index_stored_bid"`
}

// NewGenerateScoreRequest returns a new empty GenerateScoreRequest struct.
func NewGenerateScoreRequest() *GenerateScoreRequest {
	return &GenerateScoreRequest{
		K:      make([]byte, 32),
		Seed:   make([]byte, 32),
		Secret: make([]byte, 32),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (g *GenerateScoreRequest) Copy() *GenerateScoreRequest {
	k := make([]byte, len(g.K))
	seed := make([]byte, len(g.Seed))
	secret := make([]byte, len(g.Secret))

	copy(k, g.K)
	copy(seed, g.Seed)
	copy(secret, g.Secret)

	return &GenerateScoreRequest{
		K:              k,
		Seed:           seed,
		Secret:         secret,
		Round:          g.Round,
		Step:           g.Step,
		IndexStoredBid: g.IndexStoredBid,
	}
}

// MGenerateScoreRequest copies the GenerateScoreRequest structure into the Rusk equivalent.
func MGenerateScoreRequest(r *rusk.GenerateScoreRequest, f *GenerateScoreRequest) {
	copy(r.K, f.K)
	copy(r.Seed, f.Seed)
	copy(r.Secret, f.Secret)
	r.Round = f.Round
	r.Step = f.Step
	r.IndexStoredBid = f.IndexStoredBid
}

// UGenerateScoreRequest copies the Rusk GenerateScoreRequest structure into the native equivalent.
func UGenerateScoreRequest(r *rusk.GenerateScoreRequest, f *GenerateScoreRequest) {
	copy(f.K, r.K)
	copy(f.Seed, r.Seed)
	copy(f.Secret, r.Secret)
	f.Round = r.Round
	f.Step = r.Step
	f.IndexStoredBid = r.IndexStoredBid
}

// MarshalGenerateScoreRequest writes the GenerateScoreRequest struct into a bytes.Buffer.
func MarshalGenerateScoreRequest(r *bytes.Buffer, f *GenerateScoreRequest) error {
	if err := encoding.Write256(r, f.K); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Seed); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Secret); err != nil {
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
	if err := encoding.Read256(r, f.K); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Seed); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Secret); err != nil {
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
	BlindbidProof  []byte `json:"blindbid_proof"`
	Score          []byte `json:"score"`
	ProverIdentity []byte `json:"prover_identity"`
}

// NewGenerateScoreResponse returns a new empty GenerateScoreResponse struct.
func NewGenerateScoreResponse() *GenerateScoreResponse {
	return &GenerateScoreResponse{
		BlindbidProof:  make([]byte, 0),
		Score:          make([]byte, 32),
		ProverIdentity: make([]byte, 32),
	}
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (g *GenerateScoreResponse) Copy() *GenerateScoreResponse {
	proof := make([]byte, len(g.BlindbidProof))
	score := make([]byte, len(g.Score))
	identity := make([]byte, len(g.ProverIdentity))

	copy(proof, g.BlindbidProof)
	copy(score, g.Score)
	copy(identity, g.ProverIdentity)

	return &GenerateScoreResponse{
		BlindbidProof:  proof,
		Score:          score,
		ProverIdentity: identity,
	}
}

// MGenerateScoreResponse copies the GenerateScoreResponse structure into the Rusk equivalent.
func MGenerateScoreResponse(r *rusk.GenerateScoreResponse, f *GenerateScoreResponse) {
	copy(r.BlindbidProof, f.BlindbidProof)
	copy(r.Score, f.Score)
	copy(r.ProverIdentity, f.ProverIdentity)
}

// UGenerateScoreResponse copies the Rusk GenerateScoreResponse structure into the native equivalent.
func UGenerateScoreResponse(r *rusk.GenerateScoreResponse, f *GenerateScoreResponse) {
	copy(f.BlindbidProof, r.BlindbidProof)
	copy(f.Score, r.Score)
	copy(f.ProverIdentity, r.ProverIdentity)
}

// MarshalGenerateScoreResponse writes the GenerateScoreResponse struct into a bytes.Buffer.
func MarshalGenerateScoreResponse(r *bytes.Buffer, f *GenerateScoreResponse) error {
	if err := encoding.WriteVarBytes(r, f.BlindbidProof); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Score); err != nil {
		return err
	}

	return encoding.Write256(r, f.ProverIdentity)
}

// UnmarshalGenerateScoreResponse reads a GenerateScoreResponse struct from a bytes.Buffer.
func UnmarshalGenerateScoreResponse(r *bytes.Buffer, f *GenerateScoreResponse) error {
	if err := encoding.ReadVarBytes(r, &f.BlindbidProof); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Score); err != nil {
		return err
	}

	return encoding.Read256(r, f.ProverIdentity)
}

// VerifyScoreRequest is used by provisioners to ensure that a given
// blind bid proof is valid.
type VerifyScoreRequest struct {
	Proof    []byte `json:"proof"`
	Score    []byte `json:"score"`
	Seed     []byte `json:"seed"`
	ProverID []byte `json:"prover_id"`
	Round    uint64 `json:"round"`
	Step     uint32 `json:"step"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (v *VerifyScoreRequest) Copy() *VerifyScoreRequest {
	proof := make([]byte, len(v.Proof))
	score := make([]byte, len(v.Score))
	seed := make([]byte, len(v.Seed))
	proverID := make([]byte, len(v.ProverID))

	copy(proof, v.Proof)
	copy(score, v.Score)
	copy(seed, v.Seed)
	copy(proverID, v.ProverID)

	return &VerifyScoreRequest{
		Proof:    proof,
		Score:    score,
		Seed:     seed,
		ProverID: proverID,
		Round:    v.Round,
		Step:     v.Step,
	}
}

// MVerifyScoreRequest copies the VerifyScoreRequest structure into the Rusk equivalent.
func MVerifyScoreRequest(r *rusk.VerifyScoreRequest, f *VerifyScoreRequest) {
	copy(r.Proof, f.Proof)
	copy(r.Score, f.Score)
	copy(r.Seed, f.Seed)
	copy(r.ProverId, f.ProverID)
	r.Round = f.Round
	r.Step = f.Step
}

// UVerifyScoreRequest copies the Rusk VerifyScoreRequest structure into the native equivalent.
func UVerifyScoreRequest(r *rusk.VerifyScoreRequest, f *VerifyScoreRequest) {
	copy(f.Proof, r.Proof)
	copy(f.Score, r.Score)
	copy(f.Seed, r.Seed)
	copy(f.ProverID, r.ProverId)
	f.Round = r.Round
	f.Step = r.Step
}

// MarshalVerifyScoreRequest writes the VerifyScoreRequest struct into a bytes.Buffer.
func MarshalVerifyScoreRequest(r *bytes.Buffer, f *VerifyScoreRequest) error {
	if err := encoding.WriteVarBytes(r, f.Proof); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Score); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.Seed); err != nil {
		return err
	}

	if err := encoding.Write256(r, f.ProverID); err != nil {
		return err
	}

	if err := encoding.WriteUint64LE(r, f.Round); err != nil {
		return err
	}

	return encoding.WriteUint32LE(r, f.Step)
}

// UnmarshalVerifyScoreRequest reads a VerifyScoreRequest struct from a bytes.Buffer.
func UnmarshalVerifyScoreRequest(r *bytes.Buffer, f *VerifyScoreRequest) error {
	if err := encoding.ReadVarBytes(r, &f.Proof); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Score); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.Seed); err != nil {
		return err
	}

	if err := encoding.Read256(r, f.ProverID); err != nil {
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
