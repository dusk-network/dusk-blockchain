package rcudp

import (
	fountain "github.com/google/gofountain"
)

// Decoder is based on fountain.newRaptorDecoder
type Decoder struct {
	symbolAlignmentSize int
	numSourceSymbols    int
	paddingSize         int

	decoder fountain.Decoder
	decoded []byte
}

// NewDecoder creates a wrapper decoder of fountain.Decoder
func NewDecoder(numSourceSymbols, symbolAlignmentSize, transferLength, paddingSize int) *Decoder {

	c := fountain.NewRaptorCodec(numSourceSymbols, symbolAlignmentSize)
	return &Decoder{
		decoder:             c.NewDecoder(transferLength),
		symbolAlignmentSize: symbolAlignmentSize,
		numSourceSymbols:    numSourceSymbols,
		paddingSize:         paddingSize}
}

// AddBlock add a LTBlock to the a decoder.
func (d *Decoder) AddBlock(b fountain.LTBlock) []byte {
	return d.AddBlocks([]fountain.LTBlock{b})
}

// AddBlocks add a set of LTBlock to the a decoder If the object is
// reconsturcted (fully decoded), AddBlocks returns the decoded object.
func (d *Decoder) AddBlocks(blocks []fountain.LTBlock) []byte {

	if d.decoded != nil {
		return d.decoded
	}

	if d.decoder.AddBlocks(blocks) {
		out := d.decoder.Decode()
		d.decoded = out[0 : len(out)-d.paddingSize]
		return d.decoded
	}

	// Not ready yet. More blocks needed to be added
	return nil
}

// IsReady returns true, if the object is already reconstructed
func (d *Decoder) IsReady() bool {
	return len(d.decoded) > 0
}
