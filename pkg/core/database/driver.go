// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package database

import (
	"errors"
	"sort"
	"sync"
)

// Replicated from "database/sql" pkg
var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

// Register makes a database driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it returns error.
func Register(driver Driver) error {
	driversMu.Lock()
	defer driversMu.Unlock()

	if driver == nil {
		return errors.New("cannot register a nil driver")
	}

	name := driver.Name()
	if _, dup := drivers[name]; dup {
		return errors.New("duplicated driver name: " + name)
	}
	drivers[name] = driver
	return nil
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	var list []string
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

// From returns a registered Driver by name
func From(name string) (Driver, error) {
	driversMu.RLock()
	defer driversMu.RUnlock()
	if _, dup := drivers[name]; dup {
		return drivers[name], nil
	}
	return nil, errors.New("Cannot find driver with name " + name)
}
