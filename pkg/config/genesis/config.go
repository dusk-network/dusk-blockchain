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

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/keys"
)

// Config is a list of configuration parameters for generating a genesis block.
type Config struct {
	timestamp int64

	initialParticipants []keys.PublicKey
	committeeMembers    [][]byte

	initialCommitteeSize   uint
	initialBlockGenerators uint

	seed []byte

	// These are in whole units of DUSK.
	stakeValue    uint64
	bidValue      uint64
	coinbaseValue uint64

	coinbaseAmount uint
}

// NewConfig will construct a new genesis config. This function does sanity checks
// on all passed parameters, and ensures a correct genesis block can be produced
// from them.
func NewConfig(timestamp int64, initialParticipants []keys.PublicKey, committeeMembers [][]byte, initialCommitteeSize, initialBlockGenerators uint, seed []byte, stakeValue, bidValue, coinbaseValue uint64, coinbaseAmount uint) (Config, error) {
	if timestamp > time.Now().Unix() {
		return Config{}, errors.New("can not generate genesis blocks from the future")
	}

	if len(committeeMembers) > len(initialParticipants) || initialBlockGenerators > uint(len(initialParticipants)) || initialCommitteeSize > uint(len(committeeMembers)) {
		return Config{}, fmt.Errorf("can not have more consensus nodes than there are initial participants - initial: %v / provisioners: %v / intended provisioners: %v, intended generators: %v", initialParticipants, committeeMembers, initialCommitteeSize, initialBlockGenerators)
	}

	c := Config{
		timestamp:              timestamp,
		initialParticipants:    initialParticipants,
		committeeMembers:       committeeMembers,
		initialCommitteeSize:   initialCommitteeSize,
		initialBlockGenerators: initialBlockGenerators,
		seed:                   seed,
		stakeValue:             stakeValue,
		bidValue:               bidValue,
		coinbaseValue:          coinbaseValue,
		coinbaseAmount:         coinbaseAmount,
	}

	return c, nil
}
