package candidate

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/key"
)

type mockSigner struct {
	bus    *eventbus.EventBus
	pubkey []byte
}

func (m *mockSigner) Sign(header.Header) ([]byte, error) {
	return make([]byte, 33), nil
}

func (m *mockSigner) Compose(pf consensus.PacketFactory) consensus.InternalPacket {
	return pf.Create(m.pubkey, 0, 1)
}

func (m *mockSigner) Gossip(msg message.Message, id uint32) error {
	m.bus.Publish(msg.Category(), msg)
	return nil
}

func (m *mockSigner) SendInternally(topic topics.Topic, msg message.Message, id uint32) error {
	m.bus.Publish(topic, msg)
	return nil
}

// Helper for reducing generation test boilerplate
type Helper struct {
	PubKeyBLS []byte
	Bus       *eventbus.EventBus
	RBus      *rpcbus.RPCBus
	*Factory
	Generator *Generator
	*consensus.SimplePlayer
	signer consensus.Signer

	ScoreChan, CandidateChan chan message.Message
	txBatchCount             uint16
}

// NewHelper creates a Helper
func NewHelper(t *testing.T, eb *eventbus.EventBus, rpcBus *rpcbus.RPCBus, txBatchCount uint16) *Helper {
	walletKeys := key.NewKeyPair([]byte("pippo"))
	factory := NewFactory(eb, rpcBus, walletKeys.PublicKey())
	g := factory.Instantiate()
	gen := g.(*Generator)
	pubkey, _ := crypto.RandEntropy(32)
	hlp := &Helper{
		PubKeyBLS:     pubkey,
		Bus:           eb,
		RBus:          rpcBus,
		Factory:       factory,
		Generator:     gen,
		SimplePlayer:  consensus.NewSimplePlayer(),
		signer:        &mockSigner{eb, pubkey},
		ScoreChan:     make(chan message.Message, 1),
		CandidateChan: make(chan message.Message, 1),
		txBatchCount:  txBatchCount,
	}
	hlp.createResultChans()
	hlp.ProvideTransactions(t)
	return hlp
}

func (h *Helper) createResultChans() {
	scoreListener := eventbus.NewChanListener(h.ScoreChan)
	h.Bus.Subscribe(topics.Score, scoreListener)
	// Candidate messages go on the gossip topic. However, since the signer is
	// manipulated to forward gossip to its original category, we listen to
	// Candidate topic
	candidateListener := eventbus.NewChanListener(h.CandidateChan)
	h.Bus.Subscribe(topics.Candidate, candidateListener)
}

// Initialize the generator with the given round update.
func (h *Helper) Initialize(ru consensus.RoundUpdate) {
	h.Generator.Initialize(h, h.signer, ru)
	provideCertificate(h.RBus)
}

func provideCertificate(rpcBus *rpcbus.RPCBus) {
	c := make(chan rpcbus.Request, 1)
	rpcBus.Register(topics.GetLastCertificate, c)

	go func(c chan rpcbus.Request) {
		r := <-c
		buf := new(bytes.Buffer)
		cert := block.EmptyCertificate()
		message.MarshalCertificate(buf, cert)
		r.RespChan <- rpcbus.Response{*buf, nil}
	}(c)
}

// TriggerBlockGeneration creates a random ScoreEvent and triggers block generation
func (h *Helper) TriggerBlockGeneration() {
	sev := randomScoreEvent()
	h.Generator.Collect(sev)
}

// ProvideTransactions sends a set of transactions upon the request of
// the blockgenerator, standing in place of the mempool.
func (h *Helper) ProvideTransactions(t *testing.T) {
	reqChan := make(chan rpcbus.Request, 1)
	h.RBus.Register(topics.GetMempoolTxsBySize, reqChan)

	go func(reqChan chan rpcbus.Request) {
		r := <-reqChan
		txs := helper.RandomSliceOfTxs(t, h.txBatchCount)

		// Cut off the coinbase
		txs = txs[1:]
		// Encode and send
		buf := new(bytes.Buffer)
		if err := encoding.WriteVarInt(buf, uint64(len(txs))); err != nil {
			panic(err)
		}

		for _, tx := range txs {
			if err := message.MarshalTx(buf, tx); err != nil {
				panic(err)
			}
		}

		r.RespChan <- rpcbus.Response{*buf, nil}
	}(reqChan)
}

func randomScoreEvent() message.ScoreProposal {
	//we don't really care about setting a right Header here
	hdr := header.Header{
		Round:     uint64(0),
		Step:      uint8(1),
		PubKeyBLS: []byte{},
	}
	return message.MockScoreProposal(hdr)
}
