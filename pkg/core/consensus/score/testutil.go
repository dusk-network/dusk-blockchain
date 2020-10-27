package score

import (
	"bytes"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

// Helper for reducing test boilerplate
type Helper struct {
	*consensus.Emitter
	ThisSender       []byte
	ProvisionersKeys []key.Keys
	P                *user.Provisioners
	Nr               int
}

// NewHelper creates a Helper
func NewHelper(provisioners int, timeOut time.Duration) *Helper {
	p, provisionersKeys := consensus.MockProvisioners(provisioners)

	mockProxy := transactions.MockProxy{
		P:  transactions.PermissiveProvisioner{},
		BG: transactions.MockBlockGenerator{},
	}
	emitter := consensus.MockEmitter(timeOut, mockProxy)
	emitter.Keys = provisionersKeys[0]

	hlp := &Helper{

		ThisSender:       emitter.Keys.BLSPubKeyBytes,
		ProvisionersKeys: provisionersKeys,
		P:                p,
		Nr:               provisioners,
		Emitter:          emitter,
	}

	/*
		go hlp.processLastCommittee(provisionersKeys)
		go hlp.processLastCertificate()
		go hlp.processMempoolTxsBySize()
	*/

	return hlp
}

func (hlp *Helper) processLastCommittee(keys []key.Keys) {
	v := make(chan rpcbus.Request, 10)
	if err := hlp.RPCBus.Register(topics.GetLastCommittee, v); err != nil {
		panic(err)
	}

	pks := make([][]byte, 0)
	for _, pk := range keys {
		pks = append(pks, pk.BLSPubKeyBytes)
	}

	for {
		r := <-v
		com := make([][]byte, 0)
		com = append(com, pks...)
		log.WithField("len pks", len(pks)).Debug("sending mocked topics.GetLastCommittee back")
		r.RespChan <- rpcbus.NewResponse(com, nil)
	}
}

func (hlp *Helper) processLastCertificate() {
	v := make(chan rpcbus.Request, 10)
	if err := hlp.RPCBus.Register(topics.GetLastCertificate, v); err != nil {
		panic(err)
	}
	for {
		r := <-v
		buf := new(bytes.Buffer)
		cert := block.EmptyCertificate()
		err := message.MarshalCertificate(buf, cert)
		log.Debug("sending mocked topics.GetLastCertificate back")
		r.RespChan <- rpcbus.NewResponse(*buf, err)
	}
}

func (hlp *Helper) processMempoolTxsBySize() {
	v := make(chan rpcbus.Request, 10)
	if err := hlp.RPCBus.Register(topics.GetMempoolTxsBySize, v); err != nil {
		panic(err)
	}
	for {
		r := <-v
		log.Debug("sending mocked topics.GetMempoolTxsBySize back")
		r.RespChan <- rpcbus.NewResponse([]transactions.ContractCall{}, nil)
	}
}
