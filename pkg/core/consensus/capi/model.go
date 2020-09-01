package capi

import (
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/message"
)

// EventQueueJSON is used as JSON rapper for eventQueue fields
type EventQueueJSON struct {
	Round     uint64          `json:"round"`
	Step      uint8           `json:"step"`
	Message   message.Message `json:"message"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// RoundInfoJSON is used as JSON wrapper for round info fields
type RoundInfoJSON struct {
	Step      uint8     `json:"step"`
	UpdatedAt time.Time `json:"updated_at"`
	Method    string    `json:"method"`
	Name      string    `json:"name"`
}
