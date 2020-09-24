package capi

import (
	"github.com/asdine/storm/v3"
	"os"
)

var (
	stormDBInstance *StormDBInstance
)

type StormDBInstance struct {
	DB *storm.DB
}

func (bdb *StormDBInstance) Close() error {
	return bdb.DB.Close()
}

func (bdb *StormDBInstance) Delete(data interface{}) error {
	return bdb.DB.DeleteStruct(data)
}

func (bdb *StormDBInstance) Find(fieldName string, value interface{}, to interface{}) error {
	err := bdb.DB.One(fieldName, value, to)
	return err
}

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
