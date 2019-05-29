package reduction

import (
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
)

type context struct {
	handler *reductionHandler
	state   consensus.State
	timer   *consensus.Timer
}

func newCtx(handler *reductionHandler, timeout time.Duration) *context {
	return &context{
		handler: handler,
		state:   consensus.NewState(),
		timer: &consensus.Timer{
			Timeout:     timeout,
			TimeoutChan: make(chan struct{}, 1),
		},
	}
}
