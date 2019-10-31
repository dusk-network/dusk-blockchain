package score

import (
	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	"github.com/dusk-network/dusk-wallet/key"
	log "github.com/sirupsen/logrus"
)

// Factory creates a first step reduction Component
type Factory struct {
	Bus eventbus.Broker
	db  database.DB
	key.ConsensusKeys
}

// NewFactory instantiates a Factory
func NewFactory(broker eventbus.Broker, consensusKeys key.ConsensusKeys, db database.DB) *Factory {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}
	return &Factory{
		Bus:           broker,
		db:            db,
		ConsensusKeys: consensusKeys,
	}
}

// Instantiate a generation component
func (f *Factory) Instantiate() consensus.Component {
	var d, k []byte
	err := f.db.View(func(t database.Transaction) error {
		var err error
		d, k, err = t.GetBidValues()
		return err
	})

	if err != nil {
		log.WithField("process", "proof generator factory").WithError(err).Warnln("error retrieving bid values from database")
	}

	var dScalar, kScalar ristretto.Scalar
	if err := dScalar.UnmarshalBinary(d); err != nil {
		// TODO: log
	}
	if err := kScalar.UnmarshalBinary(k); err != nil {
		// TODO: log
	}

	return NewComponent(f.Bus, f.ConsensusKeys, dScalar, kScalar)
}
