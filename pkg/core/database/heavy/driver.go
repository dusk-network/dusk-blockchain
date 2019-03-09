package heavy

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

type driver struct {
}

func (d driver) Open(path string, network protocol.Magic, readonly bool) (database.DB, error) {
	return NewDatabase(path, readonly)
}

func init() {
	driver := driver{}
	database.Register(driverName, driver)
}
