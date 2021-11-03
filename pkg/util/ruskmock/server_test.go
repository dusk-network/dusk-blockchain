// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package ruskmock

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/dusk-network/bls12_381-sign/bls12_381-sign-go/bls"
	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/user"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"

	crypto "github.com/dusk-network/dusk-crypto/hash"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/stretchr/testify/assert"
)

const walletDBName = "walletDB"

func TestGetProvisioners(t *testing.T) {
	s := setupRuskMockTest(t, DefaultConfig())
	defer cleanup(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := client.CreateStateClient(ctx, "localhost:10000")

	resp, err := c.GetProvisioners(ctx, &rusk.GetProvisionersRequest{})
	assert.NoError(t, err)

	p := user.NewProvisioners()
	memberMap := make(map[string]*user.Member)

	for i := range resp.Provisioners {
		member := new(user.Member)
		transactions.UMember(resp.Provisioners[i], member)

		memberMap[string(member.PublicKeyBLS)] = member

		p.Set.Insert(member.PublicKeyBLS)
	}

	p.Members = memberMap

	// Should have gotten an identical set
	assert.Equal(t, s.p, p)
}

func TestVerifyStateTransition(t *testing.T) {
	s := setupRuskMockTest(t, DefaultConfig())
	defer cleanup(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := client.CreateStateClient(ctx, "localhost:10000")

	resp, err := c.VerifyStateTransition(ctx, &rusk.VerifyStateTransitionRequest{})
	assert.NoError(t, err)

	// Should have gotten an empty FailedCalls slice
	assert.Empty(t, resp.FailedCalls)
}

func TestFailedVerifyStateTransition(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PassStateTransitionValidation = false

	s := setupRuskMockTest(t, cfg)
	defer cleanup(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := client.CreateStateClient(ctx, "localhost:10000")

	// Send a request with 5 calls
	calls := make([]*rusk.Transaction, 5)
	for i := range calls {
		call := new(rusk.Transaction)
		tx := transactions.RandTx()

		transactions.MTransaction(call, tx)

		calls[i] = call
	}

	resp, err := c.VerifyStateTransition(ctx, &rusk.VerifyStateTransitionRequest{Txs: calls})
	assert.NoError(t, err)

	// FailedCalls should be a slice of numbers 0 to 4
	assert.Equal(t, 5, len(resp.FailedCalls))

	for i, n := range resp.FailedCalls {
		assert.Equal(t, uint64(i), n)
	}
}

func TestExecuteStateTransition(t *testing.T) {
	s := setupRuskMockTest(t, DefaultConfig())
	defer cleanup(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := client.CreateStateClient(ctx, "localhost:10000")

	// We execute the state transition with a single stake transaction, to ensure
	// the provisioners are updated.
	sc, _ := client.CreateStakeClient(ctx, "localhost:10000")
	value := uint64(712389)

	_, pk, rpk := bls.GenerateKeysWithRaw()

	tx, err := sc.NewStake(ctx, &rusk.StakeTransactionRequest{
		Value:        value,
		PublicKeyBls: pk,
	})
	assert.NoError(t, err)

	resp, err := c.ExecuteStateTransition(ctx, &rusk.ExecuteStateTransitionRequest{
		Txs:    []*rusk.Transaction{tx},
		Height: 1,
	})
	assert.NoError(t, err)

	assert.True(t, resp.Success)

	// Check that the provisioner set has a stake for our public key, with the
	// chosen value
	m := s.p.Members[string(pk)]
	assert.Equal(t, value, m.Stakes[0].Amount)

	// Ensure mapping Compressed Pk to Uncompressed Pk is correct
	rpk2 := s.p.GetRawPublicKeyBLS(pk)

	assert.True(t, bytes.Equal(rpk, rpk2))
}

func TestFailedExecuteStateTransition(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PassStateTransition = false

	s := setupRuskMockTest(t, cfg)
	defer cleanup(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := client.CreateStateClient(ctx, "localhost:10000")

	resp, err := c.ExecuteStateTransition(ctx, &rusk.ExecuteStateTransitionRequest{})
	assert.NoError(t, err)

	assert.False(t, resp.Success)
}

func TestNewStake(t *testing.T) {
	s := setupRuskMockTest(t, DefaultConfig())
	defer cleanup(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := client.CreateStakeClient(ctx, "localhost:10000")

	pk, _ := crypto.RandEntropy(96)

	resp, err := c.NewStake(ctx, &rusk.StakeTransactionRequest{
		Value:        100,
		PublicKeyBls: pk,
	})
	assert.NoError(t, err)

	// The bls public key and locktime should be included in the calldata.
	var locktime uint64

	plBuf := bytes.NewBuffer(resp.Payload)

	pl := transactions.NewTransactionPayload()
	assert.NoError(t, transactions.UnmarshalTransactionPayload(plBuf, pl))

	buf := bytes.NewBuffer(pl.CallData)

	err = encoding.ReadUint64LE(buf, &locktime)
	assert.NoError(t, err)

	assert.Equal(t, uint64(250000), locktime)

	pkBLS := make([]byte, 0)

	err = encoding.ReadVarBytes(buf, &pkBLS)
	assert.NoError(t, err)

	assert.Equal(t, pk, pkBLS[0:96])
	assert.Equal(t, uint32(4), resp.Type)
}

func setupRuskMockTest(t *testing.T, cfg *Config) *Server {
	c := config.Registry{}
	// Hardcode wallet values, so that it always starts up correctly
	c.Wallet.Store = walletDBName
	c.Wallet.File = "../../../devnet-wallets/wallet0.dat"

	s, err := New(cfg, c)
	assert.NoError(t, err)

	assert.NoError(t, s.Serve("tcp", ":10000"))
	return s
}

func cleanup(s *Server) {
	_ = s.Stop()

	if err := os.RemoveAll(walletDBName + "_2"); err != nil {
		panic(err)
	}
}
