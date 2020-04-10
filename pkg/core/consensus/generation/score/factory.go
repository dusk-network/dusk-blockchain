package score

import (
	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

// Factory creates a score.Generator.
type Factory struct {
	Bus eventbus.Broker
	db  database.DB
	key.ConsensusKeys
}

// NewFactory instantiates a Factory.
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

// Instantiate a generation component. It will attempt to retrieve the relevant values
// for score generation from the DB before instantiation, as these are ephemeral and
// can not be kept on the Factory for this reason.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	var d, k []byte
	err := f.db.View(func(t database.Transaction) error {
		var err error
		d, k, err = t.FetchBidValues()
		return err
	})

	if err != nil {
		log.WithField("process", "proof generator factory").WithError(err).Warnln("error retrieving bid values from database")
	}

	var dScalar, kScalar ristretto.Scalar
	if err := dScalar.UnmarshalBinary(d); err != nil {
		log.WithField("process", "score generator factory").WithError(err).Errorln("could not unmarshal D bytes into a scalar")
	}

	if err := kScalar.UnmarshalBinary(k); err != nil {
		log.WithField("process", "score generator factory").WithError(err).Errorln("could not unmarshal K bytes into a scalar")
	}

	return NewComponent(f.Bus, f.ConsensusKeys, dScalar, kScalar)
}
