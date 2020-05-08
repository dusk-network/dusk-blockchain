package transactor

import (
	"bytes"
	"context"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	walletdb "github.com/dusk-network/dusk-blockchain/pkg/core/data/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/wallet"
)

var testnet = byte(2)

func (t *Transactor) createFromSeed(seedBytes []byte, password string) (transactions.PublicKey, transactions.ViewKey, error) {
	var pk transactions.PublicKey
	var vk transactions.ViewKey
	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return pk, vk, err
	}

	// Then create the wallet with seed and password
	_, err = wallet.LoadFromSeed(seedBytes, testnet, db, password, cfg.Get().Wallet.File, cfg.Get().Wallet.SecretKeyFile, &t.secretKey)
	if err != nil {
		_ = db.Close()
		return pk, vk, err
	}

	return t.loadPK(t.secretKey)
}

func (t *Transactor) loadWallet(password string) (transactions.PublicKey, transactions.ViewKey, error) {
	var pk transactions.PublicKey
	var vk transactions.ViewKey
	// First load the database
	db, err := walletdb.New(cfg.Get().Wallet.Store)
	if err != nil {
		return pk, vk, err
	}

	// Then load the wallet
	w, err := wallet.LoadFromFile(testnet, db, password, cfg.Get().Wallet.File, cfg.Get().Wallet.SecretKeyFile)
	if err != nil {
		_ = db.Close()
		return pk, vk, err
	}

	// //TODO: assign wallet here still make sense ?
	// t.w = w
	return t.loadPK(w.SecretKey)
}

func (t *Transactor) loadPK(sk transactions.SecretKey) (transactions.PublicKey, transactions.ViewKey, error) {
	//get the pub key and return
	// TODO: use a parent context here
	ctx := context.Background()
	return t.keyMaster.Keys(ctx, sk)
}

// DecodeAddressToPublicKey will decode a []byte to rusk.PublicKey
func DecodeAddressToPublicKey(in []byte) (transactions.PublicKey, error) {
	var pk transactions.PublicKey
	var buf = &bytes.Buffer{}
	_, err := buf.Write(in)
	if err != nil {
		return pk, err
	}

	pk.AG = new(transactions.CompressedPoint)
	pk.BG = new(transactions.CompressedPoint)
	pk.AG.Y = make([]byte, 32)
	pk.BG.Y = make([]byte, 32)

	if _, err = buf.Read(pk.AG.Y); err != nil {
		return pk, err
	}

	if _, err = buf.Read(pk.BG.Y); err != nil {
		return pk, err
	}

	return pk, nil
}
