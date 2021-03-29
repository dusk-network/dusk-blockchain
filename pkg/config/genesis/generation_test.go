// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package genesis_test

import (
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/config/genesis"
	"github.com/stretchr/testify/assert"
)

// A generated genesis block should be entirely deterministic.
func TestEquality(t *testing.T) {
	cfg, err := genesis.GetPresetConfig("devnet")
	assert.NoError(t, err)

	b1 := genesis.Generate(cfg)
	b2 := genesis.Generate(cfg)

	assert.True(t, b1.Equals(b2))
}

/*
import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	walletdb "github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-blockchain/pkg/rpc/client"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

func TestWallets(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	keyMaster, _ := client.CreateKeysClient(ctx, "127.0.0.1:10000")
	wallets := make([]*wallet.Wallet, 120)
	pubKeys := make([]keys.PublicKey, 120)
	blsKeys := make([][]byte, 120)
	for i := 0; i < 120; i++ {
		w := generateWallet(t, i, keyMaster)
		wallets[i] = w
		pubKeys[i] = w.PublicKey
		blsKeys[i] = w.Keys().BLSPubKeyBytes
	}

	fmt.Println(pubKeys)
	fmt.Println(blsKeys)
}

func generateWallet(t *testing.T, i int, keyMaster rusk.KeysClient) *wallet.Wallet {
	seed, err := wallet.GenerateNewSeed(nil)
	if err != nil {
		t.Fatal(err)
	}

	db, err := walletdb.New("walletDB" + strconv.Itoa(i))
	if err != nil {
		t.Fatal(err)
	}

	gskr := new(rusk.GenerateKeysRequest)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sk := keys.NewSecretKey()
	pk := keys.NewPublicKey()
	vk := keys.NewViewKey()
	res, err := keyMaster.GenerateKeys(ctx, gskr)
	if err != nil {
		t.Fatal(err)
	}
	keys.USecretKey(res.Sk, sk)
	keys.UPublicKey(res.Pk, pk)
	keys.UViewKey(res.Vk, vk)

	skBuf := new(bytes.Buffer)
	if err = keys.MarshalSecretKey(skBuf, sk); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}

	keysJSON := wallet.KeysJSON{
		Seed:      seed,
		SecretKey: skBuf.Bytes(),
		PublicKey: *pk,
		ViewKey:   *vk,
	}

	w, err := wallet.LoadFromSeed(byte(2), db, "password", "wallet"+strconv.Itoa(i)+".dat", keysJSON)
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}

	_ = db.Close()
	if err := os.RemoveAll("walletDB" + strconv.Itoa(i)); err != nil {
		t.Fatal(err)
	}

	return w
}
*/
