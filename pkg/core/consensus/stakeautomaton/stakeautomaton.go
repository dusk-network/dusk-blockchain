// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package stakeautomaton

import (
	"context"
	"fmt"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	"github.com/dusk-network/dusk-protobuf/autogen/go/node"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var l = log.WithField("process", "consensus").
	WithField("actor", "StakeAutomaton")

// StakeAutomaton is a process that keeps note of when certain consensus transactions
// expire, and makes sure the node remains within the bidlist/committee, when those
// transactions are close to expiring.
type StakeAutomaton struct {
	eventBroker eventbus.Broker
	rpcBus      *rpcbus.RPCBus
	blockChan   <-chan block.Block

	height         uint64
	stakeEndHeight uint64
	running        bool
}

// How many blocks away from expiration the transactions should be
// renewed.
const renewalOffset = 100

// New creates a new instance of StakeAutomaton that is used to automate the
// resending of stakes and alleviate the burden for a user to having to
// manually manage restaking.
func New(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, srv *grpc.Server) *StakeAutomaton {
	a := &StakeAutomaton{
		eventBroker:    eventBroker,
		rpcBus:         rpcBus,
		stakeEndHeight: 1,
		running:        false,
	}

	if srv != nil {
		node.RegisterProvisionerServer(srv, a)
	}

	return a
}

// AutomateStakes will automate the sending of stakes.
func (m *StakeAutomaton) AutomateStakes(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
	if !m.running {
		// We only initialize the `roundChan` here so that we don't clog the channel with
		// blocks while the maintainer is not actually running yet.
		m.blockChan, _ = consensus.InitAcceptedBlockUpdate(m.eventBroker)
		m.running = true

		go m.Listen()
	}

	return &node.GenericResponse{Response: "stake transactions are now being automated"}, nil
}

// Listen to round updates and takes the proper decision Stake-wise.
func (m *StakeAutomaton) Listen() {
	for blk := range m.blockChan {
		m.height = blk.Header.Height + 1

		if m.height+renewalOffset >= m.stakeEndHeight {
			if err := m.sendStake(); err != nil {
				l.WithError(err).Error("could not send stake tx")
				continue
			}
		}
	}
}

func (m *StakeAutomaton) sendStake() error {
	amount, lockTime := m.getTxSettings()
	if amount == 0 || lockTime == 0 {
		return fmt.Errorf("invalid settings: amount: %v / locktime: %v", amount, lockTime)
	}

	l.WithFields(log.Fields{
		"amount":   amount,
		"locktime": lockTime,
	}).Trace("Sending stake tx")

	req := &node.StakeRequest{
		Amount:   amount,
		Fee:      config.MinFee,
		Locktime: lockTime,
	}

	timeoutSendStakeTX := time.Duration(config.Get().Timeout.TimeoutSendStakeTX) * time.Second

	_, err := m.rpcBus.Call(topics.SendStakeTx, rpcbus.NewRequest(req), timeoutSendStakeTX)
	if err != nil {
		l.WithFields(log.Fields{
			"amount":   amount,
			"locktime": lockTime,
			"err":      err,
		}).Error("error sending stake tx")
		return err
	}

	m.stakeEndHeight = lockTime + m.height
	return nil
}

func (m *StakeAutomaton) getTxSettings() (uint64, uint64) {
	settings := config.Get().Consensus
	amount := settings.DefaultAmount
	lockTime := settings.DefaultLockTime

	if lockTime > config.MaxLockTime {
		l.Warnf("default locktime exceeds maximum (%v) - defaulting to %v", lockTime, config.MaxLockTime)
		lockTime = config.MaxLockTime
	}

	// Convert amount from atomic units to whole units of DUSK
	amount = amount * wallet.DUSK

	return amount, lockTime
}
