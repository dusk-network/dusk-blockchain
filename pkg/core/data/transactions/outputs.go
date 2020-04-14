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
func (out Outputs) Less(i, j int) bool {
	return bytes.Compare(out[i].PubKey.P.Bytes(), out[j].PubKey.P.Bytes()) == -1
}

// Swap implements the sort interface
func (out Outputs) Swap(i, j int) { out[i], out[j] = out[j], out[i] }

// Equals returns true, if two slices of Outputs are the same
func (out Outputs) Equals(other Outputs) bool {
	if len(out) != len(other) {
		return false
	}

	whole := make(Outputs, len(out)*2)
	copy(whole, out)
	copy(whole[len(out):], other)

	sort.Sort(whole)

	for i := 0; i < len(whole); i += 2 {
		firstOutput := whole[i]
		secondOutput := whole[i+1]
		if !firstOutput.Equals(secondOutput) {
			return false
		}
	}
	return true
}

// HasDuplicates checks whether an output contains a duplicate
// This is done by checking that there are no matching Destination keys
func (out Outputs) HasDuplicates() bool {
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		if bytes.Equal(out[i].PubKey.P.Bytes(), out[j].PubKey.P.Bytes()) {
			return true
		}
	}
	return false
}
