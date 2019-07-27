package reduction

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
)

type context struct {
	handler *reductionHandler
	state   consensus.State
	timer   *consensus.Timer
}

func newCtx(handler *reductionHandler, timeOut time.Duration) *context {
	return &context{
		handler: handler,
		state:   consensus.NewState(),
		timer:   consensus.NewTimer(timeOut, make(chan struct{}, 1)),
	}
}
