// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package key

import (
	"github.com/dusk-network/bls12_381-sign-go/bls"
)

// Keys are the keys used during consensus.
type Keys struct {
	BLSPubKey    []byte
	BLSSecretKey []byte
}

// NewRandKeys will generate and return new bls and ed25519
// keys to be used in consensus.
func NewRandKeys() Keys {
	sk, pk := bls.GenerateKeys()

	return Keys{
		BLSPubKey:    pk,
		BLSSecretKey: sk,
	}
}
