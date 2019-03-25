package transactions

import (
	"bytes"
	"sort"
)

// Nitpick: The sort interface for input and output are similar.

// Outputs is a slice of pointers to a set of `input`'s
type Outputs []*Output

// Len implements the sort interface
func (out Outputs) Len() int { return len(out) }

// Less implements the sort interface
func (out Outputs) Less(i, j int) bool { return bytes.Compare(out[i].DestKey, out[j].DestKey) == -1 }

// Swap implements the sort interface
func (out Outputs) Swap(i, j int) { out[i], out[j] = out[j], out[i] }

// Equals returns true, if two slices of Outputs are the same
func (out Outputs) Equals(other Outputs) bool {
	// Sort both sets incase they are out of order
	sort.Sort(out)
	sort.Sort(other)

	if len(out) != len(other) {
		return false
	}

	for i := range out {
		firstInput := out[i]
		secondInput := other[i]
		if !firstInput.Equals(secondInput) {
			return false
		}
	}
	return true
}

// HasDuplicates checks whether an output contains a duplicate
// This is done by checking that there are no matching Destination keys
func (out Outputs) HasDuplicates() bool {
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		if bytes.Equal(out[i].DestKey, out[j].DestKey) {
			return true
		}
	}
	return false
}
