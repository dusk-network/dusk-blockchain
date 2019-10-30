package reduction

import (
	"bytes"
	"math"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/committee"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-wallet/key"
)

const maxCommitteeSize = 64

type (
	// Handler is responsible for performing operations that need to know
	// about specific event fields.
	Handler struct {
		*committee.Handler
	}
)

// newHandler will return a Handler, injected with the passed committee
// and an unmarshaller which uses the injected validation function.
func NewHandler(keys key.ConsensusKeys, p user.Provisioners) *Handler {
	return &Handler{
		Handler: committee.NewHandler(keys, p),
	}
}

// AmMember checks if we are part of the committee.
func (b *Handler) AmMember(round uint64, step uint8) bool {
	return b.Handler.AmMember(round, step, maxCommitteeSize)
}

func (b *Handler) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	return b.Handler.IsMember(pubKeyBLS, round, step, maxCommitteeSize)
}

func (b *Handler) VotesFor(pubKeyBLS []byte, round uint64, step uint8) int {
	return b.Handler.VotesFor(pubKeyBLS, round, step, maxCommitteeSize)
}

// Verify the BLS signature of the Reduction event. Since the payload is nil, verifying the signature equates to verifying solely the Header
func (b *Handler) VerifySignature(hdr header.Header, sig []byte) error {
	packet := new(bytes.Buffer)
	if err := header.MarshalSignableVote(packet, hdr, nil); err != nil {
		return err
	}

	return msg.VerifyBLSSignature(hdr.PubKeyBLS, packet.Bytes(), sig)
}

func (b *Handler) Quorum() int {
	return int(math.Ceil(float64(b.CommitteeSize(maxCommitteeSize)) * 0.75))
}

// Committee returns a VotingCommittee for a given round and step.
func (b *Handler) Committee(round uint64, step uint8) user.VotingCommittee {
	return b.Handler.Committee(round, step, maxCommitteeSize)
}
