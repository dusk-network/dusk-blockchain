package request

import (
	"errors"
)

// ByteToStatus attempts to convert a given byte to Status
func ByteToStatus(b byte) (Status, error) {
	switch b {
	case 0x00:
		return Success, nil
	case 0x01:
		return Failure, nil
	default:
		return UndefinedStatus, errors.New("The provided byte cannot be converted to a status")
	}
}

const (
	// Success represent a request that was executed without errors
	Success Status = 0x00
	// Failure represent a request that was executed and produced errors
	Failure Status = 0x01
	// UndefinedStatus is a byte that could not be parsed into a status
	UndefinedStatus Status = 0xff
)

// Status is the possible states of a response
type Status byte
