package database

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"testing"
)

type driverA struct{}

func (d driverA) Open(path string, network protocol.Magic, readonly bool) (DB, error) {
	return nil, nil
}

type driverB struct{}

func (d driverB) Open(path string, network protocol.Magic, readonly bool) (DB, error) {
	return nil, nil
}

func TestDuplicatedDriver(t *testing.T) {

	unregisterAllDrivers()
	err := Register("driver_a", &driverA{})
	if err != nil {
		t.Fatal("Registering DB driver failed")
	}

	err = Register("driver_a", &driverA{})
	if err == nil {
		t.Fatal("Error for duplicated driver not returned")
	}
}

func TestListDriver(t *testing.T) {

	unregisterAllDrivers()
	Register("driver_b", &driverB{})
	Register("driver_a", &driverA{})

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
	Register("driver_b", &driverB{})
	Register("driver_a", &driverA{})

	driver := From("driver_a")
	if driver == nil {
		t.Fatal("A registerd driver not found")
	}

	driver = From("driver_non")
	if driver != nil {
		t.Fatal("Invalid driver")
	}
}
