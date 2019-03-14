package reduction

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type reductionMessage interface {
	GetCommonFields() reductionBase
	IsSigSetReductionMessage() bool
}

type reductionBase struct {
	VotedHash  []byte
	SignedHash []byte
	PubKeyBLS  []byte
	Round      uint64
	Step       uint8
}

func decodeReductionBase(r io.Reader) (*reductionBase, error) {
	var votedHash []byte
	if err := encoding.Read256(r, &votedHash); err != nil {
		return nil, err
	}

	var signedHash []byte
	if err := encoding.ReadBLS(r, &signedHash); err != nil {
		return nil, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
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

	return &reductionBase{
		VotedHash:  votedHash,
		SignedHash: signedHash,
		PubKeyBLS:  pubKeyBLS,
		Round:      round,
		Step:       step,
	}, nil
}

func (r Reducer) createReductionMessage() (*bytes.Buffer, error) {
	buffer := bytes.NewBuffer(r.currentHash)

	signedHash, err := bls.Sign(r.BLSSecretKey, r.BLSPubKey, r.currentHash)
	if err != nil {
		return nil, err
	}

	if err := encoding.WriteBLS(buffer, signedHash.Compress()); err != nil {
		return nil, err
	}

	if err := encoding.WriteVarBytes(buffer, r.BLSPubKey.Marshal()); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint64(buffer, binary.LittleEndian, r.round); err != nil {
		return nil, err
	}

	if err := encoding.WriteUint8(buffer, r.step); err != nil {
		return nil, err
	}

	if r.inSigSetPhase {
		if err := encoding.Write256(buffer, r.winningBlockHash); err != nil {
			return nil, err
		}
	}

	return buffer, nil
}
