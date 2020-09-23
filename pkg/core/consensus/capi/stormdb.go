package capi

import (
	"github.com/asdine/storm/v3"
	"os"
	"sync"
)

var (
	mutex           = sync.Mutex{}
	stormDBInstance *StormDBInstance
)

type StormDBInstance struct {
	DB *storm.DB
}

func (bdb *StormDBInstance) Close() error {
	mutex.Lock()
	defer mutex.Unlock()
	return bdb.DB.Close()
}

func (bdb *StormDBInstance) Delete(data interface{}) error {
	mutex.Lock()
	defer mutex.Unlock()
	return bdb.DB.DeleteStruct(data)
}

func (bdb *StormDBInstance) Find(fieldName string, value interface{}, to interface{}) error {
	mutex.Lock()
	locked := true
	defer func() {
		if locked {
			mutex.Unlock()
		}
	}()
	err := bdb.DB.One(fieldName, value, to)
	mutex.Unlock()
	locked = false
	return err
}

func (bdb *StormDBInstance) Save(data interface{}) error {
	mutex.Lock()
	defer mutex.Unlock()
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

func NewStormDBInstance(filename string) (*StormDBInstance, error) {
	mutex.Lock()
	defer mutex.Unlock()
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
