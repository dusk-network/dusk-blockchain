package agreement

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// agreementMessage defines an agreement message on the Dusk wire protocol.
type agreementMessage struct {
	VoteSet       []*msg.Vote
	SignedVoteSet []byte
	PubKeyBLS     []byte
	Round         uint64
	Step          uint8
	BlockHash     []byte
}

type blockAgreementMessage struct {
	*agreementMessage
}

type sigSetAgreementMessage struct {
	*agreementMessage
	SigSetHash []byte
}

func decodeBlockAgreement(r io.Reader) (*blockAgreementMessage, error) {
	agreement, err := decodeAgreement(r)
	if err != nil {
		return nil, err
	}
	return &blockAgreementMessage{
		agreementMessage: agreement,
	}, nil
}

func decodeSigSetAgreement(r io.Reader) (*sigSetAgreementMessage, error) {
	agreement, err := decodeAgreement(r)
	if err != nil {
		return nil, err
	}

	var sigSetHash []byte
	if err := encoding.Read256(r, &sigSetHash); err != nil {
		return nil, err
	}

	return &sigSetAgreementMessage{
		agreementMessage: agreement,
		SigSetHash:       sigSetHash,
	}, nil
}

// DecodeAgreementMessage will decode a scoreMessage struct from r,
// and return it.
func decodeAgreement(r io.Reader) (*agreementMessage, error) {
	voteSet, err := msg.DecodeVoteSet(r)
	if err != nil {
		return nil, err
	}

	var signedVoteSet []byte
	if err := encoding.ReadBLS(r, &signedVoteSet); err != nil {
		return nil, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
		return nil, err
	}

	var blockHash []byte
	if err := encoding.Read256(r, &blockHash); err != nil {
		return nil, err
	}

	var round uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &round); err != nil {
		return nil, err
	}

	var step uint8
	if err := encoding.ReadUint8(r, &step); err != nil {
		return nil, err
	}

	return &agreementMessage{
		VoteSet:       voteSet,
		SignedVoteSet: signedVoteSet,
		PubKeyBLS:     pubKeyBLS,
		BlockHash:     blockHash,
		Round:         round,
		Step:          step,
	}, nil
}

func (a *agreementMessage) equal(other *agreementMessage) bool {
	return (bytes.Equal(a.PubKeyBLS, other.PubKeyBLS)) && (a.Round == other.Round) && (a.Step == other.Step)
}

func (b *blockAgreementMessage) equal(other *blockAgreementMessage) bool {
	return b.agreementMessage.equal(other.agreementMessage)
}

func (s *sigSetAgreementMessage) equal(other *sigSetAgreementMessage) bool {
	return s.agreementMessage.equal(other.agreementMessage) &&
		bytes.Equal(s.SigSetHash, other.SigSetHash)
}

func (s *agreementMessage) verifyVoteSetSignature() error {
	//TODO
	return nil

}

func (s *agreementMessage) verifyVoteSet() error {
	/*
		for ...
		msg.VerifyVote()
	*/
	return nil
}
