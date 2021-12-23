// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package consensus

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

type (
	// Phase is used whenever an instantiation is needed.
	Phase interface {
		// Initialize accepts as an
		// argument an interface, usually a message or the result  of the state
		// function execution. It provides the capability to create a closure of sort.
		Initialize(InternalPacket) PhaseFn
	}

	// PhaseFn represents the recursive consensus state function.
	PhaseFn interface {
		// Run the phase function.
		Run(context.Context, *Queue, chan message.Message, RoundUpdate, uint8) PhaseFn

		// String returns the description of this phase function.
		String() string
	}

	// Controller is a factory for the ControlFn. It basically relates to the
	// Agreement, which needs a different execution each round.
	Controller interface {
		// GetControlFn returns a ControlFn.
		GetControlFn() ControlFn
	}

	// ControlFn represents the asynchronous loop controlling the commencement
	// of the Phase transition.
	ControlFn func(context.Context, *Queue, <-chan message.Message, <-chan message.Message, RoundUpdate) Results
)
