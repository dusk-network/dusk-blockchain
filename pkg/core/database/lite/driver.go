package lite

import (
	"errors"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var (
	// DriverName Unique name of Lite driver
	DriverName = "Lite_v0.1.0"
)

type driver struct {
}

func (d driver) Open(path string, network protocol.Magic, readonly bool) (database.DB, error) {
	if network != protocol.DevNet {
		return nil, errors.New("lite Driver supports only DevNet")
	}
	return NewDatabase(path, readonly)
}

func (d driver) Name() string {
	return DriverName
}

func init() {
	// Register the Lite driver.
	driver := driver{}
	err := database.Register(driver)

	if err != nil {
		panic(err)
	}
}
