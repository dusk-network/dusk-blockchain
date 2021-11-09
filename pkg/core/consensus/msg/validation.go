// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package msg

import (
	"github.com/dusk-network/bls12_381-sign/bls"
)

// VerifyBLSSignature returns err if signature is invalid.
func VerifyBLSSignature(pk, signature, message []byte) error {
	apk, err := bls.CreateApk(pk)
	if err != nil {
		return err
	}

	return bls.Verify(apk, signature, message)
}
