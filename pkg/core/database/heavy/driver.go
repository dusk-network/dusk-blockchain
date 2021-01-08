// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package heavy

import (
	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	log "github.com/sirupsen/logrus"
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
	d := driver{}
	err := database.Register(&d)
	if err != nil {
		log.Panic(err)
	}
}

// CreateDBConnection creates a connection with the DB using the `heavy` driver
func CreateDBConnection() (database.Driver, database.DB) {
	drvr, err := database.From(cfg.Get().Database.Driver)
	if err != nil {
		log.Panic(err)
	}

	db, err := drvr.Open(cfg.Get().Database.Dir, protocol.MagicFromConfig(), false)
	if err != nil {
		log.Panic(err)
	}

	return drvr, db
}
