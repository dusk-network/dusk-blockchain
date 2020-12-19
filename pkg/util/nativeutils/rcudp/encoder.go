// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package rcudp

import (
	"errors"
	"math"

	fountain "github.com/google/gofountain"
)

// Encoder is wrapper around Raptor RFC5053 from gofountain.EncodeLTBlocks
type Encoder struct {

	// Raptor codes configuration
	maxPacketSize       uint16
	redundancyFactor    uint8
	SymbolAlignmentSize uint16

	// message to be encoded
	message []byte

	// output parameters
	PaddingSize      uint16
	NumSourceSymbols int
}

// NewEncoder creates  Raptor RFC5053 wrapper
func NewEncoder(message []byte, maxPacketSize uint16, redundancyFactor uint8, symbolAlignmentSize uint16) (*Encoder, error) {

	if maxPacketSize%symbolAlignmentSize != 0 {
		return nil, errors.New("the symbol size MUST be a multiple of Al")
	}

	if maxPacketSize == 0 {
		return nil, errors.New("max packat size must be positive")
	}

	minMessageLength := maxPacketSize*4 + 1
	if len(message) < int(minMessageLength) {
		maxPacketSize /= 2
	}

	return &Encoder{
		message:             message,
		maxPacketSize:       maxPacketSize,
		redundancyFactor:    redundancyFactor,
		SymbolAlignmentSize: symbolAlignmentSize}, nil
}

// GenerateBlocks encodes raptor codes
// NB: This method is destructive to the w.message array.
func (e *Encoder) GenerateBlocks() []fountain.LTBlock {

	// Add padding
	e.PaddingSize = e.calcPadding()
	paddingBytes := make([]byte, int(e.PaddingSize))
	messageWithPadding := append(e.message, paddingBytes...)

	// Encoder Codec
	numSourceSymbols, err := e.sourceBlockCount()
	if err != nil {
		panic(err)
	}
	e.NumSourceSymbols = int(numSourceSymbols)

	c := fountain.NewRaptorCodec(e.NumSourceSymbols, int(e.SymbolAlignmentSize))
	ids := generate2IDs(e.NumSourceSymbols * int(e.redundancyFactor))

	return fountain.EncodeLTBlocks(messageWithPadding, ids, c)
}

// TransferLength is the number of message plus calculated padding
func (e *Encoder) TransferLength() int {
	return len(e.message) + int(e.PaddingSize)
}

func (e *Encoder) calcPadding() uint16 {
	return uint16(e.alignedSourceBlockSize() - len(e.message))
}

func (e *Encoder) sourceBlockCount() (uint16, error) {

	alignedSize := float64(e.alignedSourceBlockSize())
	blockCount := math.Ceil(alignedSize / float64(e.maxPacketSize))

	// NumSourceSymbols = K. Must be in the range [4, 8192] (inclusive)
	if blockCount < 4 || blockCount > 8192 {
		return 0, errors.New("invalid block count")
	}

	return uint16(blockCount), nil
}

func (e *Encoder) alignedSourceBlockSize() int {
	transferLength := float64(len(e.message))
	a := math.Ceil(transferLength / float64(e.maxPacketSize))
	return int(math.Max(float64(int(e.maxPacketSize)*int(a)), float64(e.maxPacketSize*4+1)))
}

func generate2IDs(numIDs int) []int64 {
	ids := make([]int64, numIDs)
	for i := range ids {
		ids[i] = int64(i)
	}
	return ids
}
