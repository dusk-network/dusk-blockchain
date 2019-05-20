package reduction

import (
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
)

type context struct {
	*user.Keys
	handler   handler
	committee committee.Committee
	state     consensus.State
	timer     *consensus.Timer
}

func newCtx(handler handler, committee committee.Committee, keys *user.Keys, timeout time.Duration) *context {
	return &context{
		handler:   handler,
		committee: committee,
		state:     consensus.NewState(),
		Keys:      keys,
		timer: &consensus.Timer{
			Timeout:     timeout,
			TimeoutChan: make(chan struct{}, 1),
		},
	}
}
