package voting

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

func unmarshalReduction(reductionBuffer *bytes.Buffer) (*reduction, error) {
	var round uint64
	if err := encoding.ReadUint64(reductionBuffer, binary.LittleEndian, &round); err != nil {
		return nil, err
	}

	var step uint8
	if err := encoding.ReadUint8(reductionBuffer, &step); err != nil {
		return nil, err
	}

	var votedHash []byte
	if err := encoding.Read256(reductionBuffer, &votedHash); err != nil {
		return nil, err
	}

	return &reduction{
		round:     round,
		step:      step,
		votedHash: votedHash,
	}, nil
}

func (v Voter) voteReduction(br *reduction) error {
	message, err := v.createReductionMessage(br)
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
	v.eventBus.Publish(string(topics.BlockReduction), fullMessage)
	return nil
}

func (v Voter) createReductionMessage(r *reduction) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	header := &consensus.EventHeader{
		PubKeyBLS: v.Keys.BLSPubKey.Marshal(),
		Round:     r.round,
		Step:      r.step,
	}

	if err := v.Marshal(buffer, header); err != nil {
		return nil, err
	}

	if err := encoding.Write256(buffer, r.votedHash); err != nil {
		return nil, err
	}

	signedHash, err := bls.Sign(v.BLSSecretKey, v.BLSPubKey, r.votedHash)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(buffer, signedHash.Compress()); err != nil {
		return nil, err
	}

	// if we have gotten a phase update, we include the winning block hash
	// to make it viable for signature set reduction phase.
	if v.winningBlockHash != nil {
		if err := encoding.Write256(buffer, v.winningBlockHash); err != nil {
			return nil, err
		}
	}

	return buffer, nil
}
