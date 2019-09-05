package database

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestPutGet(t *testing.T) {

	path := "mainnet"

	// New
	db, err := New(path)
	assert.Nil(t, err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	// Put
	key := []byte("hello")
	value := []byte("world")
	err = db.Put(key, value)
	assert.Nil(t, err)

	// Close and re-open database
	err = db.Close()
	assert.Nil(t, err)
	db, err = New(path)
	assert.Nil(t, err)

	// Get
	val, err := db.Get(key)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(val, value))

	// Delete
	err = db.Delete(key)
	assert.Nil(t, err)

	// Get after delete
	val, err = db.Get(key)
	assert.Equal(t, leveldb.ErrNotFound, err)
	assert.True(t, bytes.Equal(val, []byte{}))
}
