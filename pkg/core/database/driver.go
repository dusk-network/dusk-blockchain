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
// it panics.
func Register(name string, driver Driver) error {
	driversMu.Lock()
	defer driversMu.Unlock()
	if _, dup := drivers[name]; dup {
		return errors.New("database: Register called twice for driver " + name)
	}
	drivers[name] = driver
	return nil
}

func unregisterAllDrivers() {
	driversMu.Lock()
	defer driversMu.Unlock()
	// For tests.
	drivers = make(map[string]Driver)
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
func From(name string) Driver {
	driversMu.RLock()
	defer driversMu.RUnlock()
	if _, dup := drivers[name]; dup {
		return drivers[name]
	}
	return nil
}
