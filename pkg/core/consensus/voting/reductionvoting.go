package voting

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

type reduction struct {
	round     uint64
	step      uint8
	votedHash []byte
}

type blockReductionCollector struct {
	blockReductionChannel chan *reduction
}

func decodeReduction(reductionBuffer *bytes.Buffer) (*reduction, error) {
	var round uint64
	if err := encoding.ReadUint64(reductionBuffer, binary.LittleEndian, &round); err != nil {
		return nil, err
	}

	var step uint8
	if err := encoding.ReadUint8(reductionBuffer, &step); err != nil {
		return nil, err
	}

	var sigSetHash []byte
	if err := encoding.Read256(reductionBuffer, &sigSetHash); err != nil {
		return nil, err
	}

	return &reduction{
		round:     round,
		step:      step,
		votedHash: sigSetHash,
	}, nil
}

func (b *blockReductionCollector) Collect(reductionBuffer *bytes.Buffer) error {
	reduction, err := decodeReduction(reductionBuffer)
	if err != nil {
		return err
	}

	b.blockReductionChannel <- reduction
	return nil
}

func (v Voter) voteBlockReduction(br *reduction) error {
	message, err := v.createBlockReductionMessage(br)
	if err != nil {
		return err
	}

	signature := ed25519.Sign(*v.EdSecretKey, message.Bytes())
	fullMessage, err := v.addPubKeyAndSig(message, signature)
	if err != nil {
		return err
	}

	// Send to wire
	v.eventBus.Publish(msg.OutgoingReductionTopic, fullMessage)
	return nil
}

func (v Voter) createBlockReductionMessage(r *reduction) (*bytes.Buffer, error) {
	buffer := bytes.NewBuffer(r.votedHash)

	signedHash, err := bls.Sign(v.BLSSecretKey, v.BLSPubKey, r.votedHash)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(buffer, signedHash.Compress()); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, v.BLSPubKey.Marshal()); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, r.round); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, r.step); err != nil {
		return nil, err
	}

	return buffer, nil
}

type sigSetReduction struct {
	*reduction
	winningBlockHash []byte
}

type sigSetReductionCollector struct {
	sigSetReductionChannel chan *sigSetReduction
}

func (b *sigSetReductionCollector) Collect(reductionBuffer *bytes.Buffer) error {
	reduction, err := decodeReduction(reductionBuffer)
	if err != nil {
		return err
	}

	var winningBlockHash []byte
	if err := encoding.Read256(reductionBuffer, &winningBlockHash); err != nil {
		return err
	}

	sigSetReduction := &sigSetReduction{
		reduction:        reduction,
		winningBlockHash: winningBlockHash,
	}

	b.sigSetReductionChannel <- sigSetReduction
	return nil
}

func (v Voter) createSigSetReductionMessage(sr *sigSetReduction) (*bytes.Buffer, error) {

	buffer, err := v.createBlockReductionMessage(sr.reduction)
	if err != nil {
		return nil, err
	}

	if err := encoding.Write256(buffer, sr.winningBlockHash); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (v Voter) voteSigSetReduction(sr *sigSetReduction) error {
	message, err := v.createSigSetReductionMessage(sr)
	if err != nil {
		return err
	}

	signature := ed25519.Sign(*v.EdSecretKey, message.Bytes())
	fullMessage, err := v.addPubKeyAndSig(message, signature)
	if err != nil {
		return err
	}

	// Send to wire
	v.eventBus.Publish(msg.OutgoingReductionTopic, fullMessage)
	return nil
}
