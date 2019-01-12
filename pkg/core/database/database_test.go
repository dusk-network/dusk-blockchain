package database_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
)

const (
	tmpDuskDir = "/.dusk"
)

var (
	basepath = os.TempDir() + tmpDuskDir
	path     = basepath + "/unittest/noded/"
)

func cleanup(db database.Database) {
	db.Close()
	os.RemoveAll(basepath)
}

func TestDbCreate(t *testing.T) {
	db, _ := database.NewDatabase(path)
	assert.NotEqual(t, nil, db)
	cleanup(db)
}

func TestDbPutGet(t *testing.T) {
	db, _ := database.NewDatabase(path)

	key := []byte("Hello")
	value := []byte("World")

	err := db.Put(key, value)
	assert.Equal(t, nil, err)

	res, err := db.Get(key)
	assert.Equal(t, nil, err)
	assert.Equal(t, value, res)
	cleanup(db)
}

func TestDbPutDelete(t *testing.T) {

	db, _ := database.NewDatabase(path)

	key := []byte("Hello")
	value := []byte("World")

	err := db.Put(key, value)

	err = db.Delete(key)
	assert.Equal(t, nil, err)

	res, err := db.Get(key)

	assert.Equal(t, errors.ErrNotFound, err)
	assert.Equal(t, res, []byte{})
	cleanup(db)
}

func TestDbHas(t *testing.T) {
	db, _ := database.NewDatabase("temp")

	res, err := db.Has([]byte("NotExist"))
	assert.Equal(t, res, false)
	assert.Equal(t, err, nil)

	key := []byte("Hello")
	value := []byte("World")

	err = db.Put(key, value)
	assert.Equal(t, nil, err)

	res, err = db.Has(key)
	assert.Equal(t, res, true)
	assert.Equal(t, err, nil)
	cleanup(db)

}

func TestDbClose(t *testing.T) {
	db, _ := database.NewDatabase("temp")
	err := db.Close()
	assert.Equal(t, nil, err)
	cleanup(db)
}
