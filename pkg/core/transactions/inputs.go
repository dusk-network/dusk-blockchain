package transactions

import (
	"bytes"
	"sort"
)

// Inputs is a slice of pointers to a set of `input`'s
type Inputs []*Input

// Len implements the sort interface
func (in Inputs) Len() int { return len(in) }

// Less implements the sort interface
func (in Inputs) Less(i, j int) bool { return bytes.Compare(in[i].KeyImage, in[j].KeyImage) == -1 }

// Swap implements the sort interface
func (in Inputs) Swap(i, j int) { in[i], in[j] = in[j], in[i] }

// Equals returns true, if two slices of inputs are the same
func (in Inputs) Equals(other Inputs) bool {
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
