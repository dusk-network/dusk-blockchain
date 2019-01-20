package database

import (
	log "github.com/sirupsen/logrus"
	cfg "github.com/spf13/viper"
	"sync"
)

var instance *BlockchainDB
var once sync.Once

// GetInstance creates a BlockchainDB instance as a Singleton
func GetInstance() *BlockchainDB {

	if instance == nil {
		once.Do(func() {
			var err error
			path := cfg.GetString("net.database.dirpath")
			instance, err = NewBlockchainDB(path)
			log.WithField("prefix", "database").Debugf("Path to database: %s", path)
			if err != nil {
				log.WithField("prefix", "database").Fatalf("Failed to find db path: %s", path)
			}
		})
	}

	return instance
}
