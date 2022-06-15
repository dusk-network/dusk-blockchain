// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package metrics

// Blocks is the placeholder for blocks.
type Blocks struct {
	Blocks []Block `json:"blocks"`
}

// Block defines a block on the Dusk blockchain.
type Block struct {
	Header Header        `json:"header"`
	Txs    []Transaction `json:"transactions"`
}

// Transaction according to the Phoenix model.
type Transaction struct {
	// Inputs  []byte `json:"inputs,omitempty"`
	// Outputs []byte `json:"outputs"`
	// Fee     *byte  `json:"fee,omitempty"`
	// Proof   []byte `json:"proof,omitempty"`
	Data []byte `json:"data,omitempty"`
}

// Header defines a block header on a Dusk block.
type Header struct {
	Version   uint8  `json:"version"`   // Block version byte
	Height    uint64 `json:"height"`    // Block height
	Timestamp string `json:"timestamp"` // Block timestamp
	GasLimit  uint64 `json:"gaslimit"`  // Block gas limit

	PrevBlockHash      []byte `json:"prev-hash"` // Hash of previous block (32 bytes)
	Seed               []byte `json:"seed"`      // Marshaled BLS signature or hash of the previous block seed (32 bytes)
	GeneratorBlsPubkey []byte `json:"generator"` // Generator BLS Public Key (96 bytes)

	//*Certificate `json:"certificate"` // Block certificate
	Hash []byte `json:"hash"` // Hash of all previous fields
}
