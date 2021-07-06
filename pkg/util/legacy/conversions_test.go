// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package legacy_test

import (
	"context"
	"encoding/binary"
	"os"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/tests/helper"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-blockchain/pkg/util/legacy"
	"github.com/dusk-network/dusk-blockchain/pkg/util/ruskmock"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/stretchr/testify/assert"
)

// Since the legacy conversions also take care to properly decode rangeproofs
// and MLSAG signatures, we can't simply use the transactions mocking package.
// This is why the RUSK mock server is used for these unit tests.

func TestStandardTxIntegrity(t *testing.T) {
	s := setupRuskMock(t)
	defer cleanup(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := client.CreateTransferClient(ctx, "localhost:10000")

	resp, err := c.NewTransfer(ctx, &rusk.TransferTransactionRequest{
		Value:     100,
		Recipient: make([]byte, 64),
	})
	assert.NoError(t, err)

	tx := transactions.NewTransaction()
	transactions.UTransaction(resp, tx)

	// Check conversion integrity
	legacyTx, err := legacy.RuskTxToTx(resp)
	assert.NoError(t, err)

	resp2, err := legacy.TxToRuskTx(legacyTx)
	assert.NoError(t, err)

	tx2 := transactions.NewTransaction()
	transactions.UTransaction(resp2, tx2)

	assert.True(t, transactions.Equal(tx, tx2))
}

func TestStakeTxIntegrity(t *testing.T) {
	s := setupRuskMock(t)
	defer cleanup(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := client.CreateStakeClient(ctx, "localhost:10000")

	resp, err := c.NewStake(ctx, &rusk.StakeTransactionRequest{
		Value:        100,
		PublicKeyBls: make([]byte, 96),
	})
	assert.NoError(t, err)

	tx := transactions.NewTransaction()
	transactions.UTransaction(resp, tx)

	// Check conversion integrity
	legacyTx, err := legacy.RuskStakeToStake(resp)
	assert.NoError(t, err)
	resp2, err := legacy.StakeToRuskStake(legacyTx)
	assert.NoError(t, err)

	tx2 := transactions.NewTransaction()
	transactions.UTransaction(resp2, tx2)

	assert.True(t, transactions.Equal(tx, tx2))
}

// Since the coinbase is a different kind of transaction, we can use the mocking
// package here. We also don't test for integrity, as we never go from legacy
// coinbases to RUSK ones - the server only receives it and never sends it.
// So instead, we just do a simple check of values.
func TestCoinbaseIntegrity(t *testing.T) {
	tx := transactions.RandDistributeTx(100, 5)
	rtx := new(rusk.Transaction)
	transactions.MTransaction(rtx, tx)
	legacyTx, err := legacy.RuskDistributeToCoinbase(rtx)
	assert.NoError(t, err)

	reward := binary.LittleEndian.Uint64(tx.Payload.CallData)
	assert.Equal(t, uint64(100), reward)
	assert.Equal(t, uint64(100), legacyTx.Rewards[0].EncryptedAmount.BigInt().Uint64())
	assert.Equal(t, tx.Payload.Notes[0].PkR, legacyTx.Rewards[0].EncryptedMask.Bytes())
}

func TestBlockIntegrity(t *testing.T) {
	s := setupRuskMock(t)
	defer cleanup(s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _ := client.CreateStakeClient(ctx, "localhost:10000")

	// Create a valid stake to put in the block.
	resp, err := c.NewStake(ctx, &rusk.StakeTransactionRequest{
		Value:        100,
		PublicKeyBls: make([]byte, 96),
	})
	assert.NoError(t, err)

	tx := transactions.NewTransaction()
	transactions.UTransaction(resp, tx)

	// Randomize block and replace txs
	blk := helper.RandomBlock(2, 5)
	blk.Txs = []transactions.ContractCall{tx}

	ob, err := legacy.NewBlockToOldBlock(blk)
	assert.NoError(t, err)

	blk2, err := legacy.OldBlockToNewBlock(ob)
	assert.NoError(t, err)

	assert.True(t, blk.Equals(blk2))
}

func setupRuskMock(t *testing.T) *ruskmock.Server {
	c := config.Registry{}
	// Hardcode wallet values, so that it always starts up correctly
	c.Wallet.Store = "walletDB"
	c.Wallet.File = "../../../devnet-wallets/wallet0.dat"
	s, err := ruskmock.New(ruskmock.DefaultConfig(), c)
	assert.NoError(t, err)
	assert.NoError(t, s.Serve("tcp", ":10000"))
	return s
}

func cleanup(s *ruskmock.Server) {
	_ = s.Stop()

	if err := os.RemoveAll("walletDB_2"); err != nil {
		panic(err)
	}
}
