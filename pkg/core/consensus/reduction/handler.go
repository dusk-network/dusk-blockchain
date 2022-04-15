// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package reduction

import (
	"bytes"
	"math"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

type (
	// Handler is responsible for performing operations that need to know
	// about specific event fields.
	Handler struct {
		*committee.Handler
	}
)

// NewHandler will return a Handler, injected with the passed committee
// and an unmarshaller which uses the injected validation function.
func NewHandler(keys key.Keys, p user.Provisioners, seed []byte) *Handler {
	return &Handler{
		Handler: committee.NewHandler(keys, p, seed),
	}
}

// AmMember checks if we are part of the committee.
func (b *Handler) AmMember(round uint64, step uint8) bool {
	return b.Handler.AmMember(round, step, config.ConsensusMaxCommitteeSize)
}

// IsMember delegates the committee.Handler to check if a BLS public key belongs
// to a committee for the specified round and step.
func (b *Handler) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	return b.Handler.IsMember(pubKeyBLS, round, step, config.ConsensusMaxCommitteeSize)
}

// VotesFor delegates the committee.Handler to accumulate Votes for the
// specified BLS public key identifying a Provisioner.
func (b *Handler) VotesFor(pubKeyBLS []byte, round uint64, step uint8) int {
	return b.Handler.VotesFor(pubKeyBLS, round, step, config.ConsensusMaxCommitteeSize)
}

// VerifySignature verifies the BLS signature of the Reduction event. Since the
// payload is nil, verifying the signature equates to verifying solely the Header.
func (b *Handler) VerifySignature(red message.Reduction) error {
	packet := new(bytes.Buffer)

	hdr := red.State()
	if err := header.MarshalSignableVote(packet, hdr); err != nil {
		return err
	}

	return msg.VerifyBLSSignature(hdr.PubKeyBLS, red.SignedHash, packet.Bytes())
}

// Quorum returns the amount of committee votes to reach a quorum.
func (b *Handler) Quorum(round uint64) int {
	return int(math.Ceil(float64(b.CommitteeSize(round, config.ConsensusMaxCommitteeSize)) * config.ConsensusQuorumThreshold))
}

// Committee returns a VotingCommittee for a given round and step.
func (b *Handler) Committee(round uint64, step uint8) user.VotingCommittee {
	return b.Handler.Committee(round, step, config.ConsensusMaxCommitteeSize)
}
