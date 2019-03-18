package transactions

import (
	"bytes"
	"sort"
)

// Nitpick: The sort interface for input and output are similar.

// Outputs is a slice of pointers to a set of `input`'s
type Outputs []*Output

// Len implements the sort interface
func (in Outputs) Len() int { return len(in) }

// Less implements the sort interface
func (in Outputs) Less(i, j int) bool { return bytes.Compare(in[i].DestKey, in[j].DestKey) == -1 }

// Swap implements the sort interface
func (in Outputs) Swap(i, j int) { in[i], in[j] = in[j], in[i] }

// Equals returns true, if two slices of Outputs are the same
func (in Outputs) Equals(other Outputs) bool {
	// Sort both sets incase they are out of order
	sort.Sort(in)
	sort.Sort(other)

	if len(in) != len(other) {
		return false
	}

	for i := range in {
		firstInput := in[i]
		secondInput := other[i]
		if !firstInput.Equals(secondInput) {
			return false
		}
	}
	return true
}
