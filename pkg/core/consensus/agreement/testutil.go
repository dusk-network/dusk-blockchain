// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package agreement

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

// Helper is a struct that facilitates sending semi-real Events with minimum effort.
type Helper struct {
	*consensus.Emitter
	P                *user.Provisioners
	Nr               int
	ProvisionersKeys []key.Keys
	Round            uint64
}

// NewHelper creates a Helper.
func NewHelper(provisioners int) *Helper {
	p, provisionersKeys := consensus.MockProvisioners(provisioners)
	emitter := consensus.MockEmitter(time.Second)
	emitter.Keys = provisionersKeys[0]

	return &Helper{
		Emitter:          emitter,
		Nr:               provisioners,
		P:                p,
		ProvisionersKeys: provisionersKeys,
		Round:            uint64(1),
	}
}

// RoundUpdate creates a valid RoundUpdate for the current round, based on the information
// passed to this Helper (i.e. round, Provisioners).
func (hlp *Helper) RoundUpdate(hash []byte) consensus.RoundUpdate {
	return consensus.RoundUpdate{
		Round: hlp.Round,
		P:     *hlp.P,
		Seed:  hash,
		Hash:  hash,
	}
}

// Spawn a number of different valid events to the Agreement component bypassing the EventBus.
func (hlp *Helper) Spawn(hash []byte) []message.Agreement {
	evs := make([]message.Agreement, hlp.Nr)
	for i := 0; i < hlp.Nr; i++ {
		evs[i] = message.MockAgreement(hash, hlp.Round, 3, hlp.ProvisionersKeys, hlp.P, i)
	}

	return evs
}
