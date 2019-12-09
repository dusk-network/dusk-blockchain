package candidate

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/generation/score"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/marshalling"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/key"
	zkproof "github.com/dusk-network/dusk-zkproof"
)

type mockSigner struct {
	bus *eventbus.EventBus
}

func (m *mockSigner) Sign(header.Header) ([]byte, error) {
	return make([]byte, 33), nil
}

func (m *mockSigner) SendAuthenticated(topic topics.Topic, hdr header.Header, b *bytes.Buffer, id uint32) error {
	m.bus.Publish(topic, b)
	return nil
}

func (m *mockSigner) SendWithHeader(topic topics.Topic, hash []byte, b *bytes.Buffer, id uint32) error {
	// Because the buffer in a BestScore message is empty, we will write the hash to it.
	// This way, we can check for correctness during tests.
	if err := encoding.Write256(b, hash); err != nil {
		return err
	}

	m.bus.Publish(topic, b)
	return nil
}

// Helper for reducing generation test boilerplate
type Helper struct {
	Bus  *eventbus.EventBus
	RBus *rpcbus.RPCBus
	*Factory
	Generator *Generator
	*consensus.SimplePlayer
	signer consensus.Signer

	ScoreChan, CandidateChan chan bytes.Buffer
	txBatchCount             uint16
}

// NewHelper creates a Helper
func NewHelper(t *testing.T, eb *eventbus.EventBus, rpcBus *rpcbus.RPCBus, txBatchCount uint16) *Helper {
	walletKeys := key.NewKeyPair([]byte("pippo"))
	factory := NewFactory(eb, rpcBus, walletKeys.PublicKey())
	g := factory.Instantiate()
	gen := g.(*Generator)
	hlp := &Helper{
		Bus:           eb,
		RBus:          rpcBus,
		Factory:       factory,
		Generator:     gen,
		SimplePlayer:  consensus.NewSimplePlayer(),
		signer:        &mockSigner{eb},
		ScoreChan:     make(chan bytes.Buffer, 1),
		CandidateChan: make(chan bytes.Buffer, 1),
		txBatchCount:  txBatchCount,
	}
	hlp.createResultChans()
	hlp.ProvideTransactions(t)
	return hlp
}

func (h *Helper) createResultChans() {
	scoreListener := eventbus.NewChanListener(h.ScoreChan)
	h.Bus.Subscribe(topics.Score, scoreListener)
	// Candidate messages go on the gossip topic
	candidateListener := eventbus.NewChanListener(h.CandidateChan)
	h.Bus.Subscribe(topics.Gossip, candidateListener)
}

// Initialize the generator with the given round update.
func (h *Helper) Initialize(ru consensus.RoundUpdate) {
	h.Generator.Initialize(h, h.signer, ru)
	provideCertificate(h.RBus)
}

func provideCertificate(rpcBus *rpcbus.RPCBus) {
	c := make(chan rpcbus.Request, 1)
	rpcBus.Register(rpcbus.GetLastCertificate, c)

	go func(c chan rpcbus.Request) {
		r := <-c
		buf := new(bytes.Buffer)
		cert := block.EmptyCertificate()
		marshalling.MarshalCertificate(buf, cert)
		r.RespChan <- rpcbus.Response{*buf, nil}
	}(c)
}

// TriggerBlockGeneration creates a random ScoreEvent and triggers block generation
func (h *Helper) TriggerBlockGeneration() {
	sev := randomScoreEvent()
	buf := new(bytes.Buffer)
	if err := score.Marshal(buf, sev); err != nil {
		panic(err)
	}

	h.Generator.Collect(consensus.Event{header.Header{}, *buf})
}

// ProvideTransactions sends a set of transactions upon the request of
// the blockgenerator, standing in place of the mempool.
func (h *Helper) ProvideTransactions(t *testing.T) {
	reqChan := make(chan rpcbus.Request, 1)
	h.RBus.Register(rpcbus.GetMempoolTxsBySize, reqChan)

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
			if err := marshalling.MarshalTx(buf, tx); err != nil {
				panic(err)
			}
		}

		r.RespChan <- rpcbus.Response{*buf, nil}
	}(reqChan)
}

func randomScoreEvent() score.Event {
	s, _ := crypto.RandEntropy(32)
	proof, _ := crypto.RandEntropy(1477)
	z, _ := crypto.RandEntropy(32)
	subset, _ := crypto.RandEntropy(32)
	seed, _ := crypto.RandEntropy(33)
	return score.Event{
		Proof: zkproof.ZkProof{
			Proof:         proof,
			Score:         s,
			Z:             z,
			BinaryBidList: subset,
		},
		Seed: seed,
	}
}
