// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/dusk-network/bls12_381-sign/go/cgo/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/header"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/block"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/stretchr/testify/require"
)

// MockEmitter is a utility to quickly wire up an emitter in tests.
func MockEmitter(consTimeout time.Duration) *Emitter {
	eb := eventbus.New()
	rpc := rpcbus.New()
	keys := key.NewRandKeys()

	return &Emitter{
		EventBus:    eb,
		RPCBus:      rpc,
		Keys:        keys,
		TimerLength: consTimeout,
	}
}

// StupidEmitter ...
func StupidEmitter() (*Emitter, *user.Provisioners) {
	committeeSize := 50
	p, provisionersKeys := MockProvisioners(committeeSize)

	emitter := MockEmitter(time.Second)
	emitter.Keys = provisionersKeys[0]

	return emitter, p
}

// MockRoundUpdate mocks a round update.
func MockRoundUpdate(round uint64, p *user.Provisioners) RoundUpdate {
	provisioners := p
	if p == nil {
		provisioners, _ = MockProvisioners(1)
	}

	seed, _ := crypto.RandEntropy(33)
	hash, _ := crypto.RandEntropy(32)

	return RoundUpdate{
		Round:           round,
		P:               *provisioners,
		Seed:            seed,
		Hash:            hash,
		LastCertificate: block.EmptyCertificate(),
	}
}

// MockProvisioners mock a Provisioner set.
func MockProvisioners(amount int) (*user.Provisioners, []key.Keys) {
	p := user.NewProvisioners()
	k := make([]key.Keys, amount)

	for i := 0; i < amount; i++ {
		keys := key.NewRandKeys()
		member := MockMember(keys)

		p.Members[string(keys.BLSPubKey)] = member
		p.Set.Insert(keys.BLSPubKey)
		k[i] = keys
	}

	return p, k
}

// MockMember mocks a Provisioner.
func MockMember(keys key.Keys) *user.Member {
	member := &user.Member{}
	member.PublicKeyBLS = keys.BLSPubKey

	var err error
	member.RawPublicKeyBLS, err = bls.PkToRaw(keys.BLSPubKey)

	if err != nil {
		panic(err)
	}

	member.Stakes = make([]user.Stake, 1)
	member.Stakes[0].Amount = 500
	member.Stakes[0].EndHeight = 10000

	return member
}

type mockPhase struct {
	callback func(ctx context.Context) bool
	packet   InternalPacket
}

func (m *mockPhase) Initialize(packet InternalPacket) PhaseFn {
	m.packet = packet
	return m
}

func (m *mockPhase) String() string {
	return "mock_step"
}

// nolint
func (m *mockPhase) Run(ctx context.Context, queue *Queue, evChan chan message.Message, r RoundUpdate, step uint8) PhaseFn {
	ctx = context.WithValue(ctx, "Packet", m.packet)
	if stop := m.callback(ctx); stop {
		return nil
	}

	return m
}

// MockPhase mocks up a consensus phase. It accepts a (recursive) function which returns a
// boolean indicating whether the consensus loop needs to return, or an error.
// If function returns true, it halts the consensus loop. An error indicates
// unrecoverable situation.
func MockPhase(cb func(ctx context.Context) bool) Phase {
	if cb == nil {
		cb = func(ctx context.Context) bool {
			return true
		}
	}

	return &mockPhase{cb, nil}
}

// TestCallback is a callback to allow for table testing based on step results.
type TestCallback func(*require.Assertions, InternalPacket, *eventbus.GossipStreamer)

// TestPhase is the phase to inject in the step under test to allow for table
// testing. It treats the packet injected through the Fn method as the result
// to test.
type TestPhase struct {
	callback TestCallback
	packet   InternalPacket
	req      *require.Assertions
	streamer *eventbus.GossipStreamer
}

// NewTestPhase returns a Phase implementation suitable for testing steps.
func NewTestPhase(t *testing.T, callback TestCallback, streamer *eventbus.GossipStreamer) *TestPhase {
	return &TestPhase{
		req:      require.New(t),
		callback: callback,
		streamer: streamer,
	}
}

func (t *TestPhase) String() string {
	return "test"
}

// Initialize is used by the step under test to provide its result.
func (t *TestPhase) Initialize(sv InternalPacket) PhaseFn {
	t.packet = sv
	return t
}

// Run does nothing else than delegating to the specified callback.
func (t *TestPhase) Run(_ context.Context, queue *Queue, _ chan message.Message, _ RoundUpdate, step uint8) PhaseFn {
	t.callback(t.req, t.packet, t.streamer)
	return nil
}

// MockNewBlockMsg ...
func MockNewBlockMsg(t *testing.T, hdr *header.Header) message.Message {
	var h header.Header
	if hdr == nil {
		h = header.Mock()
		h.Round = 1
		hash, _ := crypto.RandEntropy(32)
		h.BlockHash = hash
	} else {
		h = *hdr
	}

	// Mock candidate
	genesis := config.DecodeGenesis()
	genesis.Header.Hash = h.BlockHash
	se := message.MockNewBlock(h, *genesis)

	return message.New(topics.NewBlock, se)
}
