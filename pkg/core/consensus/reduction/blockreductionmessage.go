package reduction

import "io"

type blockReductionMessage struct {
	reductionBase
}

func decodeBlockReductionMessage(r io.Reader) (*blockReductionMessage, error) {
	reductionBase, err := decodeReductionBase(r)
	if err != nil {
		return nil, err
	}

	return &blockReductionMessage{
		reductionBase: *reductionBase,
	}, nil
}

func (b *blockReductionMessage) GetCommonFields() reductionBase {
	return b.reductionBase
}

func (b *blockReductionMessage) IsSigSetReductionMessage() bool {
	return false
}
