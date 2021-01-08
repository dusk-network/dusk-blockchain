// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package lite

import (
	"github.com/dusk-network/dusk-blockchain/pkg/core/database"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	log "github.com/sirupsen/logrus"
)

var (
	// DriverName is the unique identifier for the lite driver
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
	d := driver{}
	if err := database.Register(&d); err != nil {
		log.Panic(err)
	}
}

// CreateDBConnection creates a connection to the DB
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
