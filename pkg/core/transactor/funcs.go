package transactor

import (
	"bytes"
	"context"
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	walletdb "github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

var testnet = byte(2)

func (t *Transactor) createFromSeed(seedBytes []byte, password string) (*rusk.PublicKey, error) {
	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return nil, err
	}

	var testnet = byte(2)

	// Then create the wallet with seed and password
	_, err = wallet.LoadFromSeed(seedBytes, testnet, db, password, cfg.Get().Wallet.File, cfg.Get().Wallet.SecretKeyFile, t.secretKey)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	//get the pub key and return
	ctx := context.Background()
	keysResponse, err := t.ruskClient.Keys(ctx, t.secretKey)
	if err != nil {
		return nil, err
	}

	return keysResponse.Pk, nil
}

func (t *Transactor) loadWallet(password string) (*rusk.PublicKey, error) {
	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return nil, err
	}

	// Then load the wallet
	w, err := wallet.LoadFromFile(testnet, db, password, cfg.Get().Wallet.File, cfg.Get().Wallet.SecretKeyFile)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	//get the pub key and return
	ctx := context.Background()
	keysResponse, err := t.ruskClient.Keys(ctx, w.SecretKey())
	if err != nil {
		return nil, err
	}

	//TODO: asign wallet here still make sense ?

	t.w = w

	return keysResponse.Pk, nil
}

func DecodeAddressToPublicKey(in []byte) (*rusk.PublicKey, error) {
	var buf = &bytes.Buffer{}
	buf.Write(in)

	AG := make([]byte, 32)
	BG := make([]byte, 32)

	_, err := buf.Read(AG)
	if err != nil {
		return nil, err
	}

	_, err = buf.Read(BG)
	if err != nil {
		return nil, err
	}

	return &rusk.PublicKey{AG: &rusk.CompressedPoint{Y: AG}, BG: &rusk.CompressedPoint{Y: BG}}, nil
}
