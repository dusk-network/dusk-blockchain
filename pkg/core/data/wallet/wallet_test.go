package wallet

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/harness/tests"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"

	assert "github.com/stretchr/testify/require"
)

const dbPath = "testDb"

const seedFile = "seed.dat"
const secretFile = "key.dat"

const address = "127.0.0.1:5051"

func TestMain(m *testing.M) {

	//start rusk mock rpc server
	tests.StartMockServer(address)

	// Start all tests
	code := m.Run()

	os.Exit(code)
}

func createRPCConn(t *testing.T) (client rusk.RuskClient, conn *grpc.ClientConn) {
	assert := assert.New(t)
	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var err error
	conn, err = grpc.DialContext(dialCtx, address, grpc.WithInsecure())
	assert.NoError(err)
	return rusk.NewRuskClient(conn), conn
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
	secretKey, err := client.GenerateSecretKey(ctx, &rusk.GenerateSecretKeyRequest{B: seed})
	assert.NoError(err)

	sk := new(transactions.SecretKey)
	transactions.USecretKey(secretKey.Sk, sk)
	assert.NotNil(sk)
	assert.NotNil(sk.A.Data)
	assert.NotNil(sk.B.Data)

	w, err := New(nil, seed, netPrefix, db, "pass", seedFile, sk)
	assert.NoError(err)

	// wrong wallet password
	loadedWallet, err := LoadFromFile(netPrefix, db, "wrongPass", seedFile)
	assert.NotNil(err)
	assert.Nil(loadedWallet)

	// correct wallet password
	loadedWallet, err = LoadFromFile(netPrefix, db, "pass", seedFile)
	assert.Nil(err)

	assert.Equal(w.SecretKey.A.Data, loadedWallet.SecretKey.A.Data)
	assert.Equal(w.SecretKey.B.Data, loadedWallet.SecretKey.B.Data)

	assert.Equal(w.consensusKeys.BLSSecretKey, loadedWallet.consensusKeys.BLSSecretKey)
	assert.True(bytes.Equal(w.consensusKeys.BLSPubKeyBytes, loadedWallet.consensusKeys.BLSPubKeyBytes))
}

func TestCatchEOF(t *testing.T) {
	netPrefix := byte(1)

	client, conn := createRPCConn(t)
	defer conn.Close()

	db, err := database.New(dbPath)
	assert.Nil(t, err)
	defer os.RemoveAll(dbPath)

	defer os.Remove(seedFile)
	defer os.Remove(secretFile)

	// Generate 1000 new wallets
	for i := 0; i < 1000; i++ {
		seed, err := GenerateNewSeed(nil)
		require.Nil(t, err)

		ctx := context.Background()
		secretKey, err := client.GenerateSecretKey(ctx, &rusk.GenerateSecretKeyRequest{B: seed})
		sk := new(transactions.SecretKey)
		transactions.USecretKey(secretKey.Sk, sk)
		require.Nil(t, err)

		_, err = New(nil, seed, netPrefix, db, "pass", seedFile, sk)
		assert.Nil(t, err)
		os.Remove(seedFile)
		os.Remove(secretFile)
	}
}
