// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package wallet

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/harness/tests"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"

	assert "github.com/stretchr/testify/require"
)

const dbPath = "testDb"

const (
	seedFile   = "seed.dat"
	secretFile = "key.dat"
)

const address = "127.0.0.1:5051"

func TestMain(m *testing.M) {
	// start rusk mock rpc server
	tests.StartMockServer(address)

	// Start all tests
	code := m.Run()

	os.Exit(code)
}

func createRPCConn(t *testing.T) (client rusk.KeysClient, conn *grpc.ClientConn) {
	assert := assert.New(t)

	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var err error
	conn, err = grpc.DialContext(dialCtx, address, grpc.WithInsecure())
	assert.NoError(err)

	return rusk.NewKeysClient(conn), conn
}

func TestNewWallet(t *testing.T) {
	assert := assert.New(t)
	netPrefix := byte(1)

	db, err := database.New(dbPath)
	assert.NoError(err)

	defer os.RemoveAll(dbPath)
	defer os.Remove(seedFile)
	defer os.Remove(secretFile)

	client, conn := createRPCConn(t)
	defer conn.Close()

	seed, err := GenerateNewSeed(nil)
	assert.NoError(err)

	ctx := context.Background()
	secretKey, err := client.GenerateKeys(ctx, &rusk.GenerateKeysRequest{})
	assert.NoError(err)

	// Since the dusk-protobuf mocks currently do not fill up the scalars,
	// we will have to do it ourselves.
	require.Nil(t, fillSecretKey(secretKey.Sk))

	sk := keys.NewSecretKey()
	keys.USecretKey(secretKey.Sk, sk)
	assert.NotNil(sk)
	assert.NotNil(sk.A)
	assert.NotNil(sk.B)

	w, err := New(seed, netPrefix, db, "pass", seedFile, sk)
	assert.NoError(err)

	// wrong wallet password
	loadedWallet, err := LoadFromFile(netPrefix, db, "wrongPass", seedFile)
	assert.NotNil(err)
	assert.Nil(loadedWallet)

	// correct wallet password
	loadedWallet, err = LoadFromFile(netPrefix, db, "pass", seedFile)
	assert.Nil(err)

	assert.Equal(w.SecretKey.A, loadedWallet.SecretKey.A)
	assert.Equal(w.SecretKey.B, loadedWallet.SecretKey.B)

	assert.Equal(w.consensusKeys.BLSSecretKey, loadedWallet.consensusKeys.BLSSecretKey)
	assert.True(bytes.Equal(w.consensusKeys.BLSPubKey, loadedWallet.consensusKeys.BLSPubKey))
}

func fillSecretKey(sk *rusk.SecretKey) error {
	bs := make([]byte, 32)
	if _, err := rand.Read(bs); err != nil {
		return err
	}

	sk.A = bs
	bs2 := make([]byte, 32)

	if _, err := rand.Read(bs); err != nil {
		return err
	}

	sk.B = bs2
	return nil
}
