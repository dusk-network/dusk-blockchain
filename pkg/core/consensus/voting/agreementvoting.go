package voting

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

func unmarshalAgreement(agreementBuffer *bytes.Buffer) (*agreement, error) {
	var round uint64
	if err := encoding.ReadUint64(agreementBuffer, binary.LittleEndian, &round); err != nil {
		return nil, err
	}

	var step uint8
	if err := encoding.ReadUint8(agreementBuffer, &step); err != nil {
		return nil, err
	}

	var hash []byte
	if err := encoding.Read256(agreementBuffer, &hash); err != nil {
		return nil, err
	}

	voteSet, err := msg.DecodeVoteSet(agreementBuffer)
	if err != nil {
		return nil, err
	}

	return &agreement{
		round:   round,
		step:    step,
		hash:    hash,
		voteSet: voteSet,
	}, nil
}

func (v Voter) voteAgreement(a *agreement) error {
	message, err := v.createAgreementMessage(a)
	if err != nil {
		return err
	}

	signature := ed25519.Sign(*v.EdSecretKey, message.Bytes())
	fullMessage, err := v.addPubKeyAndSig(message, signature)
	if err != nil {
		return err
	}

	// Send to wire
	// TODO: need a specific topic for messages intended for peer
	// need to determine if its block or sigset phase when sending so
	// the peer shouldnt have to keep up with it.
	v.eventBus.Publish(string(topics.Agreement), fullMessage)
	return nil
}

func (v Voter) createAgreementMessage(a *agreement) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	header := &consensus.EventHeader{
		PubKeyBLS: v.Keys.BLSPubKey.Marshal(),
		Round:     a.round,
		Step:      a.step,
	}

	if err := v.Marshal(buffer, header); err != nil {
		return nil, err
	}

	encodedVoteSet, err := msg.EncodeVoteSet(a.voteSet)
	if err != nil {
		return nil, err
	}

	if _, err := buffer.Write(encodedVoteSet); err != nil {
		return nil, err
	}

	signedVoteSet, err := bls.Sign(v.BLSSecretKey, v.BLSPubKey, encodedVoteSet)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(buffer, signedVoteSet.Compress()); err != nil {
		return nil, err
	}

	if v.winningBlockHash == nil {
		if err := encoding.Write256(buffer, a.hash); err != nil {
			return nil, err
		}

		return buffer, nil
	}

	if err := encoding.Write256(buffer, v.winningBlockHash); err != nil {
		return nil, err
	}

	if err := encoding.Write256(buffer, a.hash); err != nil {
		return nil, err
	}

	return buffer, nil
}
