package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/database"
)

var (
	TEST = []byte("TEST")
)

func TestTableCreate(t *testing.T) {
	db, _ := database.NewDatabase(path)
	table := database.NewTable(db, TEST)

	assert.NotEqual(t, nil, table)
	cleanup(db)
}

func TestTablePutGet(t *testing.T) {
	db, _ := database.NewDatabase(path)
	table := database.NewTable(db, TEST)

	key := []byte("Hello")
	value := []byte("World")

	err := table.Put(key, value)
	assert.Equal(t, nil, err)

	res, err := table.Get(key)
	assert.Equal(t, nil, err)
	assert.Equal(t, value, res)
	cleanup(table)
}

func TestTablePutDelete(t *testing.T) {
	db, _ := database.NewDatabase(path)
	table := database.NewTable(db, TEST)

	key := []byte("Hello")
	value := []byte("World")

	err := table.Put(key, value)

	err = table.Delete(key)
	assert.Equal(t, nil, err)

	res, err := table.Get(key)

	assert.Equal(t, errors.ErrNotFound, err)
	assert.Equal(t, res, []byte{})
	cleanup(table)
}

func TestTableHas(t *testing.T) {
	db, _ := database.NewDatabase(path)
	table := database.NewTable(db, TEST)

	res, err := table.Has([]byte("NotExist"))
	assert.Equal(t, res, false)
	assert.Equal(t, err, nil)

	key := []byte("Hello")
	value := []byte("World")

	err = table.Put(key, value)
	assert.Equal(t, nil, err)

	res, err = table.Has(key)
	assert.Equal(t, res, true)
	assert.Equal(t, err, nil)
	cleanup(table)

}

func TestTableClose(t *testing.T) {
	db, _ := database.NewDatabase(path)
	table := database.NewTable(db, TEST)

	err := table.Close()
	assert.Equal(t, nil, err)
	cleanup(table)
}

func TestTableBatchGet(t *testing.T) {
	//TODO
}

func TestTableBatchDelete(t *testing.T) {
	//TODO
}
