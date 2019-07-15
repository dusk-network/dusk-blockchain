package lite

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
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
		panic(err)
	}
}

func CreateDBConnection() (database.Driver, database.DB) {
	drvr, err := database.From(DriverName)
	if err != nil {
		panic(err)
	}

	db, err := drvr.Open("", protocol.TestNet, false)
	if err != nil {
		panic(err)
	}

	return drvr, db
}
