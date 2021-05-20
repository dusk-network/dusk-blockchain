// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package bidautomaton

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

var l = log.WithField("process", "BidAutomaton")

// BidAutomaton is used to automate renewal of bids.
type BidAutomaton struct {
	eventBroker eventbus.Broker
	rpcBus      *rpcbus.RPCBus
	blockChan   <-chan block.Block

	height       uint64
	bidEndHeight uint64
	running      bool
}

// How many blocks away from expiration the transactions should be
// renewed.
const renewalOffset = 100

// New creates a new instance of the BidAutomaton. Upon request, it will start automatically
// sending bidding transactions whenever necessary to keep the node active as a block generator.
func New(eventBroker eventbus.Broker, rpcBus *rpcbus.RPCBus, srv *grpc.Server) *BidAutomaton {
	a := &BidAutomaton{
		eventBroker:  eventBroker,
		rpcBus:       rpcBus,
		bidEndHeight: 1,
		running:      false,
	}

	if srv != nil {
		node.RegisterBlockGeneratorServer(srv, a)
	}
	return a
}

// AutomateBids will automate the sending of bids.
func (m *BidAutomaton) AutomateBids(ctx context.Context, e *node.EmptyRequest) (*node.GenericResponse, error) {
	if !m.running {
		m.blockChan, _ = consensus.InitAcceptedBlockUpdate(m.eventBroker)
		m.running = true

		go m.Listen()
	}

	return &node.GenericResponse{Response: "Bid transactions are now being automated"}, nil
}

// Listen to round updates and send bids when necessary.
func (m *BidAutomaton) Listen() {
	for blk := range m.blockChan {
		// Rehydrate consensus state
		m.height = blk.Header.Height + 1

		if m.height+renewalOffset >= m.bidEndHeight {
			if err := m.sendBid(); err != nil {
				l.WithError(err).Error("could not send bid tx")
				continue
			}
		}
	}
}

// nolint
func (m *BidAutomaton) sendBid() error {
	amount, lockTime := m.getTxSettings()
	if amount == 0 || lockTime == 0 {
		return fmt.Errorf("invalid settings: amount: %v / locktime: %v", amount, lockTime)
	}

	l.WithFields(log.Fields{
		"amount":   amount,
		"locktime": lockTime,
	}).Trace("Sending bid tx")

	req := &node.BidRequest{
		Amount:   amount,
		Fee:      config.MinFee,
		Locktime: lockTime,
	}
	timeoutSendBidTX := time.Duration(config.Get().Timeout.TimeoutSendBidTX) * time.Second
	_, err := m.rpcBus.Call(topics.SendBidTx, rpcbus.NewRequest(req), timeoutSendBidTX)
	if err != nil {
		l.WithFields(log.Fields{
			"amount":   amount,
			"locktime": lockTime,
			"err":      err,
		}).Error("failed to send bid tx")
		return err
	}

	m.bidEndHeight = lockTime + m.height
	return nil
}

func (m *BidAutomaton) getTxSettings() (uint64, uint64) {
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
