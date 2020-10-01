package consensus

import (
	"bytes"
	"context"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
)

// MockEmitter is a utility to quickly wire up an emitter in tests
func MockEmitter(consTimeout time.Duration, proxy transactions.Proxy) *Emitter {
	eb := eventbus.New()
	rpc := rpcbus.New()
	keys, _ := key.NewRandKeys()
	_, pk := transactions.MockKeys()

	buf := new(bytes.Buffer)
	_ = transactions.MarshalPublicKey(buf, *pk)

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
}

func (m *mockPhase) Fn(_ InternalPacket) PhaseFn {
	return m.Run
}

func (m *mockPhase) Run(ctx context.Context, queue *Queue, evChan chan message.Message, r RoundUpdate, step uint8) (PhaseFn, error) {
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
	return &mockPhase{cb}
}
