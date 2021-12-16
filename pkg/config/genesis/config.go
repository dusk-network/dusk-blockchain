// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package genesis

import (
	"errors"
	"fmt"
	"time"
)

// PublicKey is a Phoenix public key, consisting of two JubJub points.
type PublicKey struct {
	AG []byte `json:"a_g"`
	BG []byte `json:"b_g"`
}

// Config is a list of configuration parameters for generating a genesis block.
type Config struct {
	timestamp int64

	initialParticipants []PublicKey
	committeeMembers    [][]byte

	initialCommitteeSize uint

	seed []byte

	// These are in whole units of DUSK.
	stakeValue    uint64
	coinbaseValue uint64

	coinbaseAmount uint
}

// NewConfig will construct a new genesis config. This function does sanity checks
// on all passed parameters, and ensures a correct genesis block can be produced
// from them.
func NewConfig(timestamp int64, initialParticipants []PublicKey, committeeMembers [][]byte, initialCommitteeSize uint, seed []byte, stakeValue, coinbaseValue uint64, coinbaseAmount uint) (Config, error) {
	if timestamp > time.Now().Unix() {
		return Config{}, errors.New("can not generate genesis blocks from the future")
	}

	if len(committeeMembers) > len(initialParticipants) || initialCommitteeSize > uint(len(committeeMembers)) {
		return Config{}, fmt.Errorf("can not have more consensus nodes than there are initial participants - initial: %v / provisioners: %v / intended provisioners: %v", initialParticipants, committeeMembers, initialCommitteeSize)
	}

	c := Config{
		timestamp:            timestamp,
		initialParticipants:  initialParticipants,
		committeeMembers:     committeeMembers,
		initialCommitteeSize: initialCommitteeSize,
		seed:                 seed,
		stakeValue:           stakeValue,
		coinbaseValue:        coinbaseValue,
		coinbaseAmount:       coinbaseAmount,
	}

	return c, nil
}
