// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package candidate

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

// Helper for reducing test boilerplate.
type Helper struct {
	*consensus.Emitter
	ThisSender       []byte
	ProvisionersKeys []key.Keys
	P                *user.Provisioners
	Nr               int
}

// NewHelper creates a Helper.
func NewHelper(provisioners int, timeOut time.Duration) *Helper {
	p, provisionersKeys := consensus.MockProvisioners(provisioners)

	emitter := consensus.MockEmitter(timeOut)
	emitter.Keys = provisionersKeys[0]

	hlp := &Helper{
		ThisSender:       emitter.Keys.BLSPubKeyBytes,
		ProvisionersKeys: provisionersKeys,
		P:                p,
		Nr:               provisioners,
		Emitter:          emitter,
	}

	// the wait group makes sure we force the registering of the methods to the RPCBus
	hlp.MockRPCCalls(provisionersKeys)
	return hlp
}

// MockRPCCalls makes sure that the RPCBus methods are registered.
func (hlp *Helper) MockRPCCalls(provisionersKeys []key.Keys) {
	hlp.processMempoolTxsBySize()
}

func (hlp *Helper) processMempoolTxsBySize() {
	v := make(chan rpcbus.Request, 10)
	if err := hlp.RPCBus.Register(topics.GetMempoolTxsBySize, v); err != nil {
		panic(err)
	}

	go func() {
		for {
			r := <-v

			log.Debug("sending mocked topics.GetMempoolTxsBySize back")

			r.RespChan <- rpcbus.NewResponse([]transactions.ContractCall{}, nil)
		}
	}()
}
