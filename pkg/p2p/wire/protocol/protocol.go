// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package protocol

import (
	"bytes"
	"fmt"
	"io"

	"github.com/Masterminds/semver"
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

	// VoucherNode indicates that a user is running a voucher seeder.
	VoucherNode ServiceFlag = 3
)

// NodeVer is the current node version.
// This is used only in the handshake, need to be removed.
var NodeVer = &Version{
	Major: 0,
	Minor: 4,
	Patch: 3,
}

// CurrentProtocolVersion indicates the protocol version used.
var CurrentProtocolVersion = semver.MustParse("0.1.0")

// VersionConstraintString represent the needed version.
const VersionConstraintString = "~0.1.0"

// VersionConstraint check incoming versions.
var VersionConstraint, _ = semver.NewConstraint(VersionConstraintString)

// VersionAsBuffer returns protocol version encoded as BytesBuffer.
func VersionAsBuffer() bytes.Buffer {
	buf := new(bytes.Buffer)

	if err := encoding.WriteUint16LE(buf, uint16(0)); err != nil {
		log.Panic(err)
	}

	if err := encoding.WriteUint16LE(buf, uint16(CurrentProtocolVersion.Major())); err != nil {
		log.Panic(err)
	}

	if err := encoding.WriteUint16LE(buf, uint16(CurrentProtocolVersion.Minor())); err != nil {
		log.Panic(err)
	}

	if err := encoding.WriteUint16LE(buf, uint16(CurrentProtocolVersion.Patch())); err != nil {
		log.Panic(err)
	}

	return *buf
}

// ExtractVersion extracts the version from io.Reader.
func ExtractVersion(r io.Reader) (*semver.Version, error) {
	buffer := make([]byte, 8)
	if _, err := io.ReadFull(r, buffer); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(buffer)

	reserved := uint16(0)
	if err := encoding.ReadUint16LE(buf, &reserved); err != nil {
		return nil, err
	}

	major := uint16(0)
	if err := encoding.ReadUint16LE(buf, &major); err != nil {
		return nil, err
	}

	minor := uint16(0)
	if err := encoding.ReadUint16LE(buf, &minor); err != nil {
		return nil, err
	}

	patch := uint16(0)
	if err := encoding.ReadUint16LE(buf, &patch); err != nil {
		return nil, err
	}

	version, err := semver.NewVersion(fmt.Sprintf("%d.%d.%d", major, minor, patch))
	if err != nil {
		return nil, err
	}

	return version, nil
}
