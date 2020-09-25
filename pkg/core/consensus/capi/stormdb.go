package capi

import (
	"os"

	"github.com/asdine/storm/v3"
)

var (
	stormDBInstance *StormDBInstance
)

// StormDBInstance holds db object
type StormDBInstance struct {
	DB *storm.DB
}

// Close will close db
func (bdb *StormDBInstance) Close() error {
	return bdb.DB.Close()
}

// Delete will delete data
func (bdb *StormDBInstance) Delete(data interface{}) error {
	return bdb.DB.DeleteStruct(data)
}

// Find will find data
func (bdb *StormDBInstance) Find(fieldName string, value interface{}, to interface{}) error {
	err := bdb.DB.One(fieldName, value, to)
	return err
}

// Save will save the data into db
func (bdb *StormDBInstance) Save(data interface{}) error {
	err := bdb.DB.Save(data)
	if err != nil && err == storm.ErrAlreadyExists {
		err = bdb.DB.Update(data)
	}
	return err
}

// GetStormDBInstance will return the actual db instance or nil
func GetStormDBInstance() *StormDBInstance {
	if stormDBInstance == nil {
		panic("StormDBInstance instance is nil")
	}
	return stormDBInstance
}

// SetStormDBInstance will set a store
func SetStormDBInstance(store *StormDBInstance) {
	stormDBInstance = store
}

// NewStormDBInstance creates a new db
func NewStormDBInstance(filename string) (*StormDBInstance, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		_, err := os.Create(filename)
		if err != nil {
			log.Fatalf("Failed to start DB file at %s", filename)
		}
	}

	log.Info("Opening DB: ", filename)
	db, err := storm.Open(filename)

	if err != nil {
		defer func() {
			if db != nil {
				_ = db.Close()
			}
			log.WithError(err).Fatal("Could not open DBFile: ", filename, ", error: ", err)
			os.Exit(1)
		}()
	}
	bdb := StormDBInstance{db}
	return &bdb, nil
}
