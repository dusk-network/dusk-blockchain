package reduction

import (
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
)

type context struct {
	handler   handler
	committee committee.Committee
	state     consensus.State
	timer     *consensus.Timer
}

func newCtx(handler handler, committee committee.Committee, timeout time.Duration) *context {
	return &context{
		handler:   handler,
		committee: committee,
		state:     consensus.NewState(),
		timer: &consensus.Timer{
			Timeout:     timeout,
			TimeoutChan: make(chan interface{}, 1),
		},
	}
}
