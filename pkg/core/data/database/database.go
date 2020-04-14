package database

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/txrecords"

	"github.com/bwesterb/go-ristretto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// DB encapsulates a leveldb.DB storage
type DB struct {
	storage *leveldb.DB
}

var (
	inputPrefix        = []byte{0x00}
	walletHeightPrefix = []byte{0x01}
	txRecordPrefix     = []byte{0x02}
	keyImagePrefix     = []byte{0x03}

	writeOptions = &opt.WriteOptions{NoWriteMerge: false, Sync: true}
)

// New creates an instance of DB
func New(path string) (*DB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("wallet cannot be used without database %s", err.Error())
	}
	return &DB{storage: db}, nil
}

// Put inserts a key and a value in the storage
func (db *DB) Put(key, value []byte) error {
	return db.storage.Put(key, value, nil)
}

// PutInput inserts a UTXO input (i.e. a Ristretto point) in the DB
func (db *DB) PutInput(encryptionKey []byte, pubkey ristretto.Point, amount, mask, privKey ristretto.Scalar, unlockHeight uint64) error {

	buf := &bytes.Buffer{}
	idb := &inputDB{
		amount:       amount,
		mask:         mask,
		privKey:      privKey,
		unlockHeight: unlockHeight,
	}

	if err := idb.Encode(buf); err != nil {
		return err
	}

	encryptedBytes, err := encrypt(buf.Bytes(), encryptionKey)
	if err != nil {
		return err
	}

	key := append(inputPrefix, pubkey.Bytes()...)

	return db.Put(key, encryptedBytes)
}

// RemoveInput removes a UTXO input from the DB
func (db *DB) RemoveInput(pubkey []byte, keyImage []byte) error {
	inputKey := append(inputPrefix, pubkey...)
	keyImageKey := append(keyImagePrefix, keyImage...)

	b := new(leveldb.Batch)
	b.Delete(inputKey)
	b.Delete(keyImageKey)

	return db.storage.Write(b, writeOptions)
}

// FetchInputs fetches transaction inputs amounting to the specified amount.
// Inputs are decrypted using a decryption key (view key)
func (db *DB) FetchInputs(decryptionKey []byte, amount int64) ([]*transactions.Input, int64, error) {

	var inputs []*inputDB

	var totalAmount = amount

	iter := db.storage.NewIterator(util.BytesPrefix(inputPrefix), nil)
	defer iter.Release()
	for iter.Next() {
		val := iter.Value()

		encryptedBytes := make([]byte, len(val))
		copy(encryptedBytes[:], val)

		decryptedBytes, err := decrypt(encryptedBytes, decryptionKey)
		if err != nil {
			return nil, 0, err
		}
		idb := &inputDB{}

		buf := bytes.NewBuffer(decryptedBytes)
		err = idb.Decode(buf)
		if err != nil {
			return nil, 0, err
		}

		// Only add unlocked inputs
		if idb.unlockHeight == 0 {
			inputs = append(inputs, idb)

			// Check if we need more inputs
			totalAmount = totalAmount - idb.amount.BigInt().Int64()
			if totalAmount <= 0 {
				break
			}
		}
	}

	if totalAmount > 0 {
		return nil, 0, errors.New("accumulated value of all of your inputs do not account for the total amount inputted")
	}

	err := iter.Error()
	if err != nil {
		return nil, 0, err
	}

	var changeAmount int64
	if totalAmount < 0 {
		changeAmount = -totalAmount
	}

	// convert inputDb to transaction input
	var tInputs []*transactions.Input
	for _, input := range inputs {
		tInputs = append(tInputs, transactions.NewInput(input.amount, input.mask, input.privKey))
	}

	return tInputs, changeAmount, nil
}

// FetchBalance calculates the balance
func (db *DB) FetchBalance(decryptionKey []byte) (uint64, uint64, error) {
	var unlockedBalance ristretto.Scalar
	unlockedBalance.SetZero()
	var lockedBalance ristretto.Scalar
	lockedBalance.SetZero()

	iter := db.storage.NewIterator(util.BytesPrefix(inputPrefix), nil)
	defer iter.Release()
	for iter.Next() {
		val := iter.Value()

		encryptedBytes := make([]byte, len(val))
		copy(encryptedBytes[:], val)

		decryptedBytes, err := decrypt(encryptedBytes, decryptionKey)
		if err != nil {
			return 0, 0, err
		}
		idb := &inputDB{}

		buf := bytes.NewBuffer(decryptedBytes)
		err = idb.Decode(buf)
		if err != nil {
			return 0, 0, err
		}

		if idb.unlockHeight == 0 {
			unlockedBalance.Add(&unlockedBalance, &idb.amount)
			continue
		}

		lockedBalance.Add(&lockedBalance, &idb.amount)
	}

	err := iter.Error()
	if err != nil {
		return 0, 0, err
	}

	return unlockedBalance.BigInt().Uint64(), lockedBalance.BigInt().Uint64(), nil
}

// UpdateLockedInputs will set the lockheight for an input to 0 if the
// given `height` is greater or equal than the input lockheight,
// signifying that this input is unlocked.
func (db *DB) UpdateLockedInputs(decryptionKey []byte, height uint64) error {
	iter := db.storage.NewIterator(util.BytesPrefix(inputPrefix), nil)
	defer iter.Release()
	for iter.Next() {
		val := iter.Value()

		encryptedBytes := make([]byte, len(val))
		copy(encryptedBytes[:], val)

		decryptedBytes, err := decrypt(encryptedBytes, decryptionKey)
		if err != nil {
			return err
		}
		idb := &inputDB{}

		buf := bytes.NewBuffer(decryptedBytes)
		err = idb.Decode(buf)
		if err != nil {
			return err
		}

		if idb.unlockHeight != 0 && idb.unlockHeight <= height {
			idb.unlockHeight = 0
			// Overwrite input
			buf := new(bytes.Buffer)
			if err := idb.Encode(buf); err != nil {
				return err
			}

			encryptedBytes, err := encrypt(buf.Bytes(), decryptionKey)
			if err != nil {
				return err
			}

			if err := db.Put(iter.Key(), encryptedBytes); err != nil {
				return err
			}
		}
	}

	return iter.Error()
}

// GetWalletHeight returns the height of the blockchain known to the wallet
func (db *DB) GetWalletHeight() (uint64, error) {
	heightBytes, err := db.storage.Get(walletHeightPrefix, nil)
	if err != nil {
		return 0, err
	}

	height := binary.LittleEndian.Uint64(heightBytes)
	return height, nil
}

// UpdateWalletHeight updates the wallet height
func (db *DB) UpdateWalletHeight(newHeight uint64) error {
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, newHeight)
	return db.Put(walletHeightPrefix, heightBytes)
}

// Get a value using a key
func (db *DB) Get(key []byte) ([]byte, error) {
	return db.storage.Get(key, nil)
}

// Delete an entry identified by the specified key
func (db *DB) Delete(key []byte) error {
	return db.storage.Delete(key, nil)
}

// Close the storage
func (db *DB) Close() error {
	return db.storage.Close()
}

// FetchTxRecords transaction records
func (db *DB) FetchTxRecords() ([]txrecords.TxRecord, error) {
	records := make([]txrecords.TxRecord, 0)
	iter := db.storage.NewIterator(util.BytesPrefix(txRecordPrefix), nil)
	defer iter.Release()

	for iter.Next() {
		// record is the key without the prefix
		val := iter.Key()[1:]

		bs := make([]byte, len(val))
		copy(bs[:], val)

		txRecord := txrecords.TxRecord{}

		if err := txrecords.Decode(bytes.NewBuffer(bs), &txRecord); err != nil {
			return nil, err
		}

		records = append(records, txRecord)
	}

	err := iter.Error()
	return records, err
}

// PutTxRecord saves the transaction record on the DB
func (db *DB) PutTxRecord(tx transactions.Transaction, direction txrecords.Direction, privView *key.PrivateView) error {
	// Schema
	//
	// key: txRecordPrefix + record
	// value: 0
	buf := new(bytes.Buffer)
	height, err := db.GetWalletHeight()
	if err != nil {
		return err
	}

	txRecord := txrecords.New(tx, height, direction, privView)
	if err := txrecords.Encode(buf, txRecord); err != nil {
		return err
	}

	key := make([]byte, 0)
	key = append(key, txRecordPrefix...)
	key = append(key, buf.Bytes()...)

	value := make([]byte, 1)
	return db.Put(key, value)
}

// PutKeyImage saves the transaction's key image used to prevent double
// spending
func (db *DB) PutKeyImage(keyImage []byte, outputKey []byte) error {
	key := append(keyImagePrefix, keyImage...)
	return db.Put(key, outputKey)
}

// GetPubKey returns the public key retrieved through the specifie key image
func (db *DB) GetPubKey(keyImage []byte) ([]byte, error) {
	key := append(keyImagePrefix, keyImage...)
	return db.Get(key)
}

// Clear all information from the database.
func (db *DB) Clear() error {
	iter := db.storage.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		if err := db.Delete(iter.Key()); err != nil {
			return err
		}
	}

	return iter.Error()
}
