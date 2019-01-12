package core

import (
	cnf "github.com/spf13/viper"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"sync"
)

var instance *Blockchain
var once sync.Once

// GetBcInstance creates a Blockchain instance as a Singleton
func GetBcInstance() (*Blockchain, error) {
	var err error

	if instance == nil {
		once.Do(func() {
			instance, err = NewBlockchain(protocol.Magic(cnf.GetInt("net.magic")))
		})
	}
	if err != nil {
		return nil, err
	}
	return instance, nil
}
