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

type Driver struct {
}

func (d Driver) Open(path string, network protocol.Magic, readonly bool) (database.DB, error) {
	if network != protocol.DevNet {
		return nil, errors.New("lite Driver supports only DevNet")
	}
	return NewDatabase(path, readonly)
}

func (d Driver) Name() string {
	return DriverName
}
