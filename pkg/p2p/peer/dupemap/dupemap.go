package dupemap

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

var obsoleteMessageRound uint64 = 3

type DupeMap struct {
	roundChan <-chan uint64 // Will get notification about new rounds in order...
	tmpMap    *TmpMap       // ...to clean up the duplicate message blacklist
}

func NewDupeMap(eventbus *wire.EventBus) *DupeMap {
	roundChan := consensus.InitRoundUpdate(eventbus)
	tmpMap := NewTmpMap(obsoleteMessageRound)
	return &DupeMap{
		roundChan,
		tmpMap,
	}
}

func (d *DupeMap) CleanOnRound() {
	for {
		round := <-d.roundChan
		d.tmpMap.UpdateHeight(round)
	}
}

func (d *DupeMap) CanFwd(payload *bytes.Buffer) bool {
	found := d.tmpMap.HasAnywhere(payload)
	if found {
		return false
	}
	d.tmpMap.Add(payload)
	return true
}
