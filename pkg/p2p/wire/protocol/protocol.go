package protocol

import (
	"fmt"
	"strings"

	cfg "gitlab.dusk.network/dusk-core/dusk-go/pkg/config"
)

// ServiceFlag indicates the services provided by the Node.
type ServiceFlag uint64

const (
	// FullNode indicates that a user is running the full node implementation of Dusk
	FullNode ServiceFlag = 1

	// LightNode indicates that a user is running a Dusk light node
	// LightNode ServiceFlag = 2 // Not implemented
)

// NodeVer is the current node version.
var NodeVer = &Version{
	Major: 0,
	Minor: 1,
	Patch: 0,
}

// Magic is the network that Dusk is running on
type Magic uint32

const (
	// MainNet identifies the production network of the Dusk blockchain
	MainNet Magic = 0x7630401f
	// TestNet identifies the test network of the Dusk blockchain
	TestNet Magic = 0x74746e41
	// DevNet identifies the development network of the Dusk blockchain
	DevNet Magic = 0x74736e40
)

// MagicFromConfig reads the loaded magic config and tries to map it to magic
// identifier. Panic, if no match found.
func MagicFromConfig() Magic {

	magic := cfg.Get().General.Network
	switch strings.ToLower(magic) {
	case "devnet":
		return DevNet
	case "testnet":
		return TestNet
	case "mainnet":
		return MainNet
	}

	// An invalid network identifier might cause node unexpected  behaviour
	panic(fmt.Sprintf("not a valid network: %s", magic))
}
