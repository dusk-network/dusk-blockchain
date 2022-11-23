// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package message

// Metadata is a struct containing messages metadata.
type Metadata struct {
	KadcastHeight byte
	Source        string
	NumNodes      uint32
}

func (m simple) Metadata() *Metadata {
	return m.metadata
}
