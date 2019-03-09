package lite

import (
	"errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

type driver struct {
}

func (d driver) Open(path string, network protocol.Magic, readonly bool) (database.DB, error) {
	if network != protocol.DevNet {
		return nil, errors.New("Lite Driver supports only DevNet")
	}
	return NewDatabase(path, readonly)
}

func init() {
	// Register the Lite driver.
	driver := driver{}
	database.Register(driverName, driver)
}
