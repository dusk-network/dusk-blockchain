package reduction

import (
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type sigSetReductionMessage struct {
	reductionBase
	WinningBlockHash []byte
}

func decodeSigSetReductionMessage(r io.Reader) (*sigSetReductionMessage, error) {
	reductionBase, err := decodeReductionBase(r)
	if err != nil {
		return nil, err
	}

	var winningBlockHash []byte
	if err := encoding.Read256(r, &winningBlockHash); err != nil {
		return nil, err
	}

	return &sigSetReductionMessage{
		reductionBase:    *reductionBase,
		WinningBlockHash: winningBlockHash,
	}, nil
}

func (s *sigSetReductionMessage) GetCommonFields() reductionBase {
	return s.reductionBase
}

func (s *sigSetReductionMessage) IsSigSetReductionMessage() bool {
	return true
}
