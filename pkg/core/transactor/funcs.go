// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package transactor

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
)

// DecodeAddressToPublicKey will decode a []byte to rusk.PublicKey.
func DecodeAddressToPublicKey(in []byte) (*keys.StealthAddress, error) {
	pk := keys.NewStealthAddress()
	buf := &bytes.Buffer{}

	_, err := buf.Write(in)
	if err != nil {
		return pk, err
	}

	pk.RG = make([]byte, 32)
	pk.PkR = make([]byte, 32)

	if _, err = buf.Read(pk.RG); err != nil {
		return pk, err
	}

	if _, err = buf.Read(pk.PkR); err != nil {
		return pk, err
	}

	return pk, nil
}
