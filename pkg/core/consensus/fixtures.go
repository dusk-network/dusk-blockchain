package consensus

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/require"
)

// MockEmitter is a utility to quickly wire up an emitter in tests
func MockEmitter(consTimeout time.Duration, proxy transactions.Proxy) *Emitter {
	eb := eventbus.New()
	rpc := rpcbus.New()
	keys, _ := key.NewRandKeys()
	_, pk := transactions.MockKeys()

	buf := new(bytes.Buffer)
	err := transactions.MarshalPublicKey(buf, *pk)
	if err != nil {
		panic("MockEmitter transactions.MarshalPublicKey")
	}

	return &Emitter{
		EventBus:    eb,
		RPCBus:      rpc,
		Keys:        keys,
		Proxy:       proxy,
		PubkeyBuf:   *buf,
		TimerLength: consTimeout,
	}
}

// MockRoundUpdate mocks a round update
func MockRoundUpdate(round uint64, p *user.Provisioners) RoundUpdate {
	var provisioners = p
	if p == nil {
		provisioners, _ = MockProvisioners(1)
	}

	seed, _ := crypto.RandEntropy(33)
	hash, _ := crypto.RandEntropy(32)
	return RoundUpdate{
		Round: round,
		P:     *provisioners,
		Seed:  seed,
		Hash:  hash,
	}
}

// MockProvisioners mock a Provisioner set
func MockProvisioners(amount int) (*user.Provisioners, []key.Keys) {
	p := user.NewProvisioners()

	k := make([]key.Keys, amount)
	for i := 0; i < amount; i++ {
		keys, _ := key.NewRandKeys()
		member := MockMember(keys)

		p.Members[string(keys.BLSPubKeyBytes)] = member
		p.Set.Insert(keys.BLSPubKeyBytes)
		k[i] = keys
	}
	return p, k
}

// MockMember mocks a Provisioner
func MockMember(keys key.Keys) *user.Member {
	member := &user.Member{}
	member.PublicKeyBLS = keys.BLSPubKeyBytes
	member.Stakes = make([]user.Stake, 1)
	member.Stakes[0].Amount = 500
	member.Stakes[0].EndHeight = 10000
	return member
}

type mockPhase struct {
	callback func(ctx context.Context) (bool, error)
	packet   InternalPacket
}

func (m *mockPhase) Fn(packet InternalPacket) PhaseFn {
	m.packet = packet
	return m.Run
}

// nolint
func (m *mockPhase) Run(ctx context.Context, queue *Queue, evChan chan message.Message, r RoundUpdate, step uint8) (PhaseFn, error) {
	ctx = context.WithValue(ctx, "Packet", m.packet)
	if stop, err := m.callback(ctx); err != nil {
		return nil, err
	} else if stop {
		return nil, nil
	}
	return m.Run, nil
}

// MockPhase mocks up a consensus phase. It accepts a (recursive) function which returns a
// boolean indicating whether the consensus loop needs to return, or an error.
// If function returns true, it halts the consensus loop. An error indicates
// unrecoverable situation
func MockPhase(cb func(ctx context.Context) (bool, error)) Phase {
	if cb == nil {
		cb = func(ctx context.Context) (bool, error) {
			return true, nil
		}
	}
	return &mockPhase{cb, nil}
}

// TestCallback is a callback to allow for table testing based on step results
type TestCallback func(*require.Assertions, InternalPacket) error

// TestPhase is the phase to inject in the step under test to allow for table
// testing. It treats the packet injected through the Fn method as the result
// to test
type TestPhase struct {
	callback TestCallback
	packet   InternalPacket
	req      *require.Assertions
}

// NewTestPhase returns a Phase implementation suitable for testing steps
func NewTestPhase(t *testing.T, callback TestCallback) *TestPhase {
	return &TestPhase{
		req:      require.New(t),
		callback: callback,
	}
}

// Fn is used by the step under test to provide its result
func (t *TestPhase) Fn(sv InternalPacket) PhaseFn {
	t.packet = sv
	return t.Run
}

// Run does nothing else than delegating to the specified callback
func (t *TestPhase) Run(_ context.Context, queue *Queue, _ chan message.Message, _ RoundUpdate, _ uint8) (PhaseFn, error) {
	return nil, t.callback(t.req, t.packet)
}

// MockScoreMsg ...
func MockScoreMsg(t *testing.T, hdr *header.Header) message.Message {
	var h header.Header
	if hdr == nil {
		h = header.Mock()
		h.Round = 1
		hash, _ := crypto.RandEntropy(32)
		h.BlockHash = hash
	} else {
		h = *hdr
	}

	// mock candidate
	genesis := config.DecodeGenesis()
	cert := block.EmptyCertificate()
	c := message.MakeCandidate(genesis, cert)

	se := message.MockScore(h, c)
	return message.New(topics.Score, se)
}
