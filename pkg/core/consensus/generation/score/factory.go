package score

import (
	"context"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus"
	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/key"
	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database/heavy"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"
	log "github.com/sirupsen/logrus"
)

// Factory creates a score.Generator.
type Factory struct {
	Bus eventbus.Broker
	db  database.DB
	key.Keys
	bg  transactions.BlockGenerator
	ctx context.Context
}

// NewFactory instantiates a Factory.
func NewFactory(ctx context.Context, broker eventbus.Broker, consensusKeys key.Keys, db database.DB, proxy transactions.Proxy) *Factory {
	if db == nil {
		_, db = heavy.CreateDBConnection()
	}
	return &Factory{
		Bus:  broker,
		db:   db,
		Keys: consensusKeys,
		ctx:  ctx,
		bg:   proxy.BlockGenerator(),
	}
}

// Instantiate a generation component. It will attempt to retrieve the relevant values
// for score generation from the DB before instantiation, as these are ephemeral and
// can not be kept on the Factory for this reason.
// Implements consensus.ComponentFactory.
func (f *Factory) Instantiate() consensus.Component {
	var d, k, edPk []byte
	err := f.db.View(func(t database.Transaction) error {
		var err error
		d, k, edPk, err = t.FetchBidValues()
		return err
	})

	if err != nil {
		log.WithField("process", "proof generator factory").WithError(err).Warnln("error retrieving bid values from database")
	}

	return NewComponent(f.ctx, f.Bus, f.Keys, d, k, edPk, f.bg)
}
