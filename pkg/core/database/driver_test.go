// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package database

import (
	"testing"
)

func unregisterAllDrivers() {
	driversMu.Lock()
	defer driversMu.Unlock()

	// For tests.
	drivers = make(map[string]Driver)
}

// Dummy DriverA.
type driverA struct{}

func (d driverA) Open(path string, readonly bool) (DB, error) {
	return nil, nil
}

func (d driverA) Name() string {
	return "driver_a"
}

func (d driverA) Close() error {
	return nil
}

// Dummy DriverB.
type driverB struct{}

func (d driverB) Open(path string, readonly bool) (DB, error) {
	return nil, nil
}

func (d driverB) Name() string {
	return "driver_b"
}

func (d driverB) Close() error {
	return nil
}

func TestDuplicatedDriver(t *testing.T) {
	unregisterAllDrivers()

	err := Register(&driverA{})
	if err != nil {
		t.Fatal("Registering DB driver failed")
	}

	err = Register(&driverA{})
	if err == nil {
		t.Fatal("Error for duplicated driver not returned")
	}

	if len(Drivers()) != 1 {
		t.Fatal("The second registering should fail")
	}
}

func TestListDriver(t *testing.T) {
	unregisterAllDrivers()

	err := Register(&driverB{})
	if err != nil {
		t.Fatal("Registering DB driverB failed")
	}

	err = Register(&driverA{})

	if err != nil {
		t.Fatal("Registering DB driverA failed")
	}

	allDrivers := Drivers()

	if allDrivers[0] != "driver_a" {
		t.Fatal("Missing a registered driver")
	}

	if allDrivers[1] != "driver_b" {
		t.Fatal("Missing a registered driver")
	}
}

func TestRetrieveDriver(t *testing.T) {
	unregisterAllDrivers()

	err := Register(&driverB{})
	if err != nil {
		t.Fatal("Registering DB driverB failed")
	}

	err = Register(&driverA{})

	if err != nil {
		t.Fatal("Registering DB driverA failed")
	}

	driver, err := From("driver_a")
	if driver == nil || err != nil {
		t.Fatal("A registered driver not found")
	}

	driver, err = From("driver_non")
	if driver != nil || err == nil {
		t.Fatal("Invalid driver")
	}
}
