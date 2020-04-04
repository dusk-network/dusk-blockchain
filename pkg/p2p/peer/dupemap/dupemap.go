package dupemap

import (
	"bytes"
)

var defaultTolerance uint64 = 3

// TODO: DupeMap should deal with value bytes.Buffer rather than pointers as it is not supposed to mutate the struct
//nolint:golint
type DupeMap struct {
	round     uint64
	tmpMap    *TmpMap
	tolerance uint64
}

// NewDupeMap returns a DupeMap
func NewDupeMap(round uint64) *DupeMap {
	tmpMap := NewTmpMap(defaultTolerance)
	return &DupeMap{
		round,
		tmpMap,
		defaultTolerance,
	}
}

// UpdateHeight for a round
func (d *DupeMap) UpdateHeight(round uint64) {
	d.tmpMap.UpdateHeight(round)
}

// SetTolerance for a round
func (d *DupeMap) SetTolerance(roundNr uint64) {
	threshold := d.tmpMap.Height() - roundNr
	d.tmpMap.DeleteBefore(threshold)
	d.tmpMap.SetTolerance(roundNr)
}

// CanFwd payload
func (d *DupeMap) CanFwd(payload *bytes.Buffer) bool {
	found := d.tmpMap.HasAnywhere(payload)
	if found {
		return false
	}
	return !d.tmpMap.Add(payload)
}
