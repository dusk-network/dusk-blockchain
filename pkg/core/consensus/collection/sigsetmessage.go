package collection

import "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

type sigSetMessage struct {
	WinningBlockHash []byte
	SigSet           []*msg.Vote
	SignedSigSet     []byte
	PubKeyBLS        []byte
	Round            uint64
	Step             uint8
}
