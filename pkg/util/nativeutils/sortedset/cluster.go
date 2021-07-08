// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package sortedset

import (
	"encoding/json"
	"fmt"

	"github.com/dusk-network/dusk-blockchain/pkg/util"
)

// Cluster is a sortedset that keeps track of duplicates.
type Cluster struct {
	Set
	elements map[string]int
}

// NewCluster returns a new empty Cluster.
func NewCluster() Cluster {
	return Cluster{
		Set:      New(),
		elements: make(map[string]int),
	}
}

// Equal tests for equality with another Cluster.
func (c Cluster) Equal(other Cluster) bool {
	for i, k := range c.Set {
		if k.Cmp(other.Set[i]) != 0 {
			return false
		}

		if c.OccurrencesOf(k.Bytes()) != other.OccurrencesOf(k.Bytes()) {
			return false
		}
	}

	return true
}

// TotalOccurrences returns the amount of elements in the cluster.
func (c Cluster) TotalOccurrences() int {
	size := 0
	for _, v := range c.elements {
		size += v
	}

	return size
}

// Unravel creates a sorted array of []byte adding as many duplicates as the
// occurrence of the various elements in ascending order.
func (c Cluster) Unravel() [][]byte {
	pks := make([][]byte, 0)

	for _, pk := range c.Set {
		occurrences := c.elements[string(pk.Bytes())]
		for i := 0; i < occurrences; i++ {
			pks = append(pks, pk.Bytes())
		}
	}

	return pks
}

// OccurrencesOf return the occurrence a []byte has been inserted in the
// cluster.
func (c Cluster) OccurrencesOf(b []byte) int {
	if n, ok := c.elements[string(b)]; ok {
		return n
	}

	return 0
}

// Insert a []byte into the cluster and updates the element counts.
// Returns true if the element is new, false otherwise.
func (c *Cluster) Insert(b []byte) bool {
	k := string(b)

	if ok := c.Set.Insert(b); !ok {
		c.elements[k] = c.elements[k] + 1
		return false
	}

	c.elements[k] = 1
	return true
}

// RemoveAll occurrences of a []byte from the cluster and updates the element counts.
// Returns the amount of occurrences that have been removed.
func (c *Cluster) RemoveAll(b []byte) int {
	k := string(b)

	occurrences, ok := c.elements[k]
	if !ok {
		return 0
	}

	c.Set.Remove(b)
	delete(c.elements, k)
	return occurrences
}

// Remove a []byte from the cluster and updates the element counts.
// Returns false if the element cannot be found.
func (c *Cluster) Remove(b []byte) bool {
	k := string(b)

	occurrences, ok := c.elements[k]
	if !ok {
		return false
	}

	if occurrences == 1 {
		c.Set.Remove(b)
		delete(c.elements, k)

		return true
	}

	c.elements[k] = occurrences - 1
	return true
}

// IntersectCluster performs an intersect operation with a Cluster represented
// through a uint64 bitmap.
func (c *Cluster) IntersectCluster(committeeSet uint64) Cluster {
	set := c.Intersect(committeeSet)

	elems := make(map[string]int)
	for _, elem := range set {
		elems[string(elem.Bytes())] = c.OccurrencesOf(elem.Bytes())
	}

	return Cluster{
		Set:      set,
		elements: elems,
	}
}

// Format implements fmt.Formatter interface.
func (c Cluster) Format(f fmt.State, r rune) {
	for _, elem := range c.Set {
		count := c.OccurrencesOf(elem.Bytes())
		r := fmt.Sprintf("(blsPk: %s,count: %d)", util.StringifyBytes(elem.Bytes()), count)
		_, _ = f.Write([]byte(r))
	}
}

// MarshalJSON ...
func (c Cluster) MarshalJSON() ([]byte, error) {
	data := make([]string, 0)

	for _, elem := range c.Set {
		count := c.OccurrencesOf(elem.Bytes())
		r := fmt.Sprintf("Key: %s, Count: %d", util.StringifyBytes(elem.Bytes()), count)
		data = append(data, r)
	}

	return json.Marshal(data)
}
