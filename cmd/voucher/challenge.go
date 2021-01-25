// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

// Secret is a shared secret between the voucher seeder and the node.
var Secret = os.Getenv("SEEDER_KEY")

// hashChallenge will hash a string using sha256.
func hashChallenge(challenge string) string {
	sha256Local := sha256.New()
	_, _ = sha256Local.Write([]byte(challenge))
	return strings.ToUpper(hex.EncodeToString(sha256Local.Sum(nil)))
}

// HashesMatch hashes a string and compares it with a provided value.
func HashesMatch(providedHash, challenge string) bool {
	computedChallenge := ComputeChallenge(challenge)
	result := computedChallenge == providedHash
	return result
}

// ComputeChallenge returns the expected answer by hashing the random string with the shared secret.
func ComputeChallenge(gen string) string {
	return hashChallenge(gen + Secret)
}
