package dupemap

import (
	"bytes"
)

var defaultTolerance uint64 = 3

type DupeMap struct {
	round     uint64
	tmpMap    *TmpMap
	tolerance uint64
}

func NewDupeMap(round uint64) *DupeMap {
	tmpMap := NewTmpMap(defaultTolerance)
	return &DupeMap{
		round,
		tmpMap,
		defaultTolerance,
	}
}

func (d *DupeMap) UpdateHeight(round uint64) {
	d.tmpMap.UpdateHeight(round)
}

func (d *DupeMap) SetTolerance(roundNr uint64) {
	threshold := d.tmpMap.Height() - roundNr
	d.tmpMap.DeleteBefore(threshold)
	d.tmpMap.SetTolerance(roundNr)
}

func (d *DupeMap) CanFwd(payload *bytes.Buffer) bool {
	found := d.tmpMap.HasAnywhere(payload)
	if found {
		return false
	}
	d.tmpMap.Add(payload)
	return true
}
