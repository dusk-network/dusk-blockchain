package voting

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

func (v Voter) voteBlockAgreement(voteSetBytes []byte) error {
	message, err := v.createBlockAgreementMessage(voteSetBytes)
	if err != nil {
		return err
	}

	signature := ed25519.Sign(*v.EdSecretKey, message.Bytes())
	fullMessage, err := v.addPubKeyAndSig(message, signature)
	if err != nil {
		return err
	}

	// Send to wire
	v.eventBus.Publish(msg.OutgoingAgreementTopic, fullMessage)
	return nil
}

func (v Voter) createBlockAgreementMessage(voteSetBytes []byte) (*bytes.Buffer, error) {
	buffer := bytes.NewBuffer(voteSetBytes)

	signedVoteSet, err := bls.Sign(v.BLSSecretKey, v.BLSPubKey, voteSetBytes)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(buffer, signedVoteSet.Compress()); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, v.BLSPubKey.Marshal()); err != nil {
		return nil, err
	}

	if v.inSigSetPhase {
		if err := encoding.Write256(buffer, v.winningBlockHash); err != nil {
			return nil, err
		}
	} else {
		if err := encoding.Write256(buffer, v.currentHash); err != nil {
			return nil, err
		}
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, v.round); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, v.step); err != nil {
		return nil, err
	}

	if v.inSigSetPhase {
		if err := encoding.Write256(buffer, v.currentHash); err != nil {
			return nil, err
		}
	}

	return buffer, nil
}
