package database

import (
	"bytes"
	"os"
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
)

const path = "mainnet"

//TODO: #446 , shall this be refactored ?
func TestPutGet(t *testing.T) {
	assert := assert.New(t)

	// New
	db, err := New(path)
	assert.NoError(err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	// Put
	key := []byte("hello")
	value := []byte("world")
	err = db.Put(key, value)
	assert.NoError(err)

	// Close and re-open database
	err = db.Close()
	assert.NoError(err)
	db, err = New(path)
	assert.NoError(err)

	// Get
	val, err := db.Get(key)
	assert.NoError(err)
	assert.True(bytes.Equal(val, value))

	// Delete
	err = db.Delete(key)
	assert.NoError(err)

	// Get after delete
	val, err = db.Get(key)
	assert.Equal(leveldb.ErrNotFound, err)
	assert.True(bytes.Equal(val, []byte{}))
}

func TestPutFetchTxRecord(t *testing.T) {
	// FIXME: 496
}

func TestPutTxRecord(t *testing.T) {
	// FIXME: 496
}

func TestClear(t *testing.T) {
	// New
	db, err := New(path)
	assert.Nil(t, err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	// Put
	key := []byte("hello")
	value := []byte("world")
	err = db.Put(key, value)
	assert.NoError(t, err)

	// Empty out database
	assert.NoError(t, db.Clear())

	// Info should now be gone entirely
	_, err = db.Get([]byte("hello"))
	assert.Error(t, err)
}
