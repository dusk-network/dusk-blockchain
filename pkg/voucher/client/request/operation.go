package request

import (
	"errors"
)

// ByteToOperation attempts to convert a given byte to Operation
func ByteToOperation(b byte) (Operation, error) {
	switch b {
	case 0x00:
		return PingOperation, nil
	case 0x01:
		return RegisterNodeOperation, nil
	case 0x02:
		return RequestSeedersOperation, nil
	case 0x03:
		return RegisterNodeVerifyChallengeOperation, nil
	default:
		return UndefinedOperation, errors.New("The provided byte cannot be converted to a operation")
	}
}

const (
	// PingOperation is a simple ping command request
	PingOperation Operation = 0x00
	// RegisterNodeOperation will make the current node available to act as seeder
	RegisterNodeOperation Operation = 0x01
	// RequestSeedersOperation will fetch seeders sockets from a voucher
	RequestSeedersOperation Operation = 0x02
	// RegisterNodeVerifyChallengeOperation will sbumit the proof the the owner of the node
	RegisterNodeVerifyChallengeOperation Operation = 0x03
	// UndefinedOperation is a byte that could not be parsed into an operation
	UndefinedOperation Operation = 0xff
)

// Operation is the possible commands to be sent to a voucher. All the operations must be synchronized with the dusk-voucher lib. Check the vendors
type Operation byte
