// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package capi

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/sortedset"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

// EventQueueJSON is used as JSON rapper for eventQueue fields.
type EventQueueJSON struct {
	ID        int              `storm:"id,increment" json:"id"` // primary key with auto increment
	Round     uint64           `json:"round"`
	Step      uint8            `json:"step"`
	Message   *message.Message `json:"message"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// RoundInfoJSON is used as JSON wrapper for round info fields.
type RoundInfoJSON struct {
	ID        int       `storm:"id,increment" json:"id"`
	Round     uint64    `storm:"index" json:"round"`
	Step      uint8     `json:"step"`
	UpdatedAt time.Time `json:"updated_at"`
	Method    string    `json:"method"`
	Name      string    `json:"name"`
}

// PeerJSON is used as JSON wrapper for peer info fields.
type PeerJSON struct {
	ID       int       `storm:"id,increment" json:"id"`
	Address  string    `json:"address"`
	Type     string    `storm:"index" json:"type"`
	Method   string    `storm:"index" json:"method"`
	LastSeen time.Time `storm:"index" json:"last_seen"`
}

// Count is the struct used to return a count for a API service.
type Count struct {
	Count int `json:"count"`
}

// PeerCount is the struct used to save a peer or remove from a PeerCount collection in the API monitoring database.
type PeerCount struct {
	ID       string    `storm:"id" json:"id"`
	LastSeen time.Time `storm:"index" json:"last_seen"`
}

// Member is the holder of bls and stakes.
type Member struct {
	PublicKeyBLS []byte  `json:"bls_key"`
	Stakes       []Stake `json:"stakes"`
}

// Stake represents the Provisioner's stake.
type Stake struct {
	Value       uint64 `json:"value"`
	Reward      uint64 `json:"reward"`
	Counter     uint64 `json:"counter"`
	Eligibility uint64 `json:"eligibility"`
}

// ProvisionerJSON represents the Provisioner.
type ProvisionerJSON struct {
	ID      uint64        `storm:"id" json:"id"`
	Set     sortedset.Set `json:"set"`
	Members []*Member     `json:"members"`
}
