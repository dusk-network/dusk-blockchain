package reduction

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
)

type (
	consensusState struct {
		Round uint64
		Step  uint8
	}

	timer struct {
		timeout     time.Duration
		timeoutChan chan interface{}
	}

	context struct {
		handler           handler
		committee         committee.Committee
		state             *consensusState
		reductionVoteChan chan *bytes.Buffer
		agreementVoteChan chan *bytes.Buffer
		timer             *timer
	}
)

func newCtx(handler handler, committee committee.Committee, timeout time.Duration) *context {
	return &context{
		handler:           handler,
		committee:         committee,
		state:             &consensusState{},
		reductionVoteChan: make(chan *bytes.Buffer, 1),
		agreementVoteChan: make(chan *bytes.Buffer, 1),
		timer: &timer{
			timeout:     timeout,
			timeoutChan: make(chan interface{}, 1),
		},
	}
}
