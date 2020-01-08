package lite

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	log "github.com/sirupsen/logrus"
)

var (
	// DriverName is the unique identifier for the heavy driver
	DriverName = "lite_v0.1.0"
)

type driver struct {
}

func (d *driver) Open(path string, network protocol.Magic, readonly bool) (database.DB, error) {
	return NewDatabase(path, network, readonly)
}

func (d *driver) Close() error {
	return nil
}

func (d *driver) Name() string {
	return DriverName
}

func init() {
	driver := driver{}
	err := database.Register(&driver)
	if err != nil {
		log.Panic(err)
	}
}

func CreateDBConnection() (database.Driver, database.DB) {
	drvr, err := database.From(DriverName)
	if err != nil {
		log.Panic(err)
	}

	db, err := drvr.Open("", protocol.TestNet, false)
	if err != nil {
		log.Panic(err)
	}

	return drvr, db
}
