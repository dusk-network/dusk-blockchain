package heavy

import (
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
)

var (
	// DriverName is the unique identifier for the heavy driver
	DriverName = "heavy_v0.1.0"
)

type driver struct {
}

func (d *driver) Open(path string, network protocol.Magic, readonly bool) (database.DB, error) {
	return NewDatabase(path, network, readonly)
}

func (d *driver) Close() error {
	return closeStorage()
}

func (d *driver) Name() string {
	return DriverName
}

func init() {
	driver := driver{}
	err := database.Register(&driver)
	if err != nil {
		panic(err)
	}
}

func CreateDBConnection() (database.Driver, database.DB) {
	drvr, err := database.From(cfg.Get().Database.Driver)
	if err != nil {
		panic(err)
	}

	db, err := drvr.Open(cfg.Get().Database.Dir, protocol.MagicFromConfig(), false)
	if err != nil {
		panic(err)
	}

	return drvr, db
}
