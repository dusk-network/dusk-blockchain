// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package selection

import (
	"bytes"

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
	return b.Handler.AmMember(round, step, config.ConsensusSelectionMaxCommitteeSize)
}

// IsMember delegates the committee.Handler to check if a BLS public key belongs
// to a committee for the specified round and step.
func (b *Handler) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	return b.Handler.IsMember(pubKeyBLS, round, step, config.ConsensusSelectionMaxCommitteeSize)
}

// VerifySignature verifies the BLS signature of the NewBlock event. Since the
// payload is nil, verifying the signature equates to verifying solely the Header.
func (b *Handler) VerifySignature(scr message.NewBlock) error {
	packet := new(bytes.Buffer)

	hdr := scr.State()
	if err := header.MarshalSignableVote(packet, hdr); err != nil {
		return err
	}

	return msg.VerifyBLSSignature(hdr.PubKeyBLS, scr.SignedHash, packet.Bytes())
}

// Committee returns a VotingCommittee for a given round and step.
func (b *Handler) Committee(round uint64, step uint8) user.VotingCommittee {
	return b.Handler.Committee(round, step, config.ConsensusSelectionMaxCommitteeSize)
}
