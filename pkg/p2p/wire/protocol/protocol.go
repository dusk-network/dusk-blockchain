// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	cfg "github.com/dusk-network/dusk-blockchain/pkg/config"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	log "github.com/sirupsen/logrus"
)

// ServiceFlag indicates the services provided by the Node.
type ServiceFlag uint64

const (
	// FullNode indicates that a user is running the full node implementation of Dusk.
	FullNode ServiceFlag = 1

	// LightNode indicates that a user is running a Dusk light node.
	// LightNode ServiceFlag = 2 // Not implemented.
)

// NodeVer is the current node version.
var NodeVer = &Version{
	Major: 0,
	Minor: 4,
	Patch: 1,
}

// Magic is the network that Dusk is running on.
type Magic uint8

const (
	// MainNet identifies the production network of the Dusk blockchain.
	MainNet Magic = iota
	// TestNet identifies the test network of the Dusk blockchain.
	TestNet
	// DevNet identifies the development network of the Dusk blockchain.
	DevNet
)

const (
	mainnetUint32 uint32 = 0x7630401f
	testnetUint32 uint32 = 0x74746e41
	//nolint
	devnetUint32    uint32 = 0x74736e40
	stressnetUint32 uint32 = 0x74726e39
)

type magicObj struct {
	Magic
	buf bytes.Buffer
	str string
}

var magics = [...]magicObj{
	{MainNet, asBuffer(0x7630401f), "mainnet"},
	{TestNet, asBuffer(0x74746e41), "testnet"},
	{DevNet, asBuffer(0x74736e40), "devnet"},
}

// Len returns the amount of bytes of the Magic sequence.
func (m Magic) Len() int {
	return magics[m].buf.Len()
}

// String representation of Magic.
func (m Magic) String() string {
	return magics[m].str
}

// ToBuffer returns the buffer representation of the Magic.
func (m Magic) ToBuffer() bytes.Buffer {
	return magics[m].buf
}

func fromUint32(n uint32) Magic {
	switch n {
	case mainnetUint32:
		return MainNet
	case testnetUint32:
		return TestNet
	default:
		return DevNet
	}
}

func asBuffer(magic uint32) bytes.Buffer {
	buf := new(bytes.Buffer)
	if err := encoding.WriteUint32LE(buf, magic); err != nil {
		log.Panic(err)
	}
	return *buf
}

// MagicFromConfig reads the loaded magic config and tries to map it to magic
// identifier. Panic, if no match found.
func MagicFromConfig() Magic {
	magic := cfg.Get().General.Network
	mstr := strings.ToLower(magic)

	for _, m := range magics {
		if mstr == m.str {
			return m.Magic
		}
	}

	// An invalid network identifier might cause node unexpected behavior.
	log.Panic(fmt.Sprintf("not a valid network: %s", magic))
	return 0
}

// Extract the magic from io.Reader. In case of unknown Magic, it returns DevNet.
func Extract(r io.Reader) (Magic, error) {
	buffer := make([]byte, 4)
	// `ReadFull` is used here, as using a plain `Read` call from a net.Conn could
	// result in the read finishing before the buffer is filled. Using `ReadFull`
	// prevents unintended 'magic mismatch' errors.
	if _, err := io.ReadFull(r, buffer); err != nil {
		return Magic(byte(255)), err
	}

	magic := binary.LittleEndian.Uint32(buffer)
	return fromUint32(magic), nil
}
