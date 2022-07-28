// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package reduction

import (
	"context"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	log "github.com/sirupsen/logrus"
)

const (
	// Republish flag instructs AsyncSend to republish the output message.
	Republish = iota
	// ValidateOnly flag instrucst AsyncSend to validate only.
	ValidateOnly
)

// AsyncSend is wrapper of SendReduction to call it in an asynchronous manner.
type AsyncSend struct {
	*Reduction

	round     uint64
	step      uint8
	candidate *block.Block
}

// NewAsyncSend ...
func NewAsyncSend(r *Reduction, round uint64, step uint8, candidate *block.Block) *AsyncSend {
	return &AsyncSend{
		Reduction: r,
		round:     round,
		step:      step,
		candidate: candidate,
	}
}

// Go executes SendReduction in a separate goroutine.
// Returns cancel func for canceling the started job.
func (a AsyncSend) Go(ctx context.Context, resp chan message.Message, flags int) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer a.recover()

		m, _, err := a.SendReduction(ctx, a.round, a.step, a.candidate)
		if err != nil {
			return
		}

		if flags == Republish {
			select {
			case resp <- m:
			default:
			}

			if err := a.Republish(m); err != nil {
				panic(err)
			}
		}
	}()

	return cancel
}

func (a AsyncSend) recover() {
	defer func() {
		if r := recover(); r != nil {
			log.WithField("round", a.round).WithField("step", a.step).
				WithError(fmt.Errorf("%+v", r)).
				Errorln("sending reduction err")
		}
	}()
}
