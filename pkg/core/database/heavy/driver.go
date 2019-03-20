package heavy

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

var (
	// DriverName Unique name of Heavy driver
	DriverName = "heavy_v0.1.0"
)

type driver struct {
}

// Open creates goleveldb storage on first call and reuse it until
// driver.Close() call. See also openStorage
func (d *driver) Open(path string, network protocol.Magic, readonly bool) (database.DB, error) {
	return NewDatabase(path, network, readonly)
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
