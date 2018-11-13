package generator

import (
	"encoding/binary"

	"github.com/toghrulmaharramov/dusk-go/ristretto"
)

// This package will generate the generators for the pedersens and the bulletproof

type Generator struct {
	data  []byte
	Bases []ristretto.Point
	i     uint64
}

// New will generate a generator which
// will use data to generate `n` points
func New(data []byte) *Generator {
	return &Generator{
		data:  data,
		Bases: []ristretto.Point{},
		i:     0,
	}
}

// Reset will set i back to zero
// NOTE: Do not use this on the same Pedersen commitment
func (g *Generator) Reset() {
	g.i = 0
}

//Clear will clear all of the Bases
// but leave the counter as is
func (g *Generator) Clear() {
	g.Bases = []ristretto.Point{}
}

// Iterate will generate a new point using
// `data` and current index `i`
func (g *Generator) Iterate() ristretto.Point {

	p := ristretto.Point{}

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(g.i))
	input := append(g.data, b...)

	p.Derive(input)

	g.i++

	return p
}

// Compute will generate num amount of points, which will act as point generators
// using the initial data.
func (g *Generator) Compute(num uint8) {
	bases := []ristretto.Point{}

	for i := uint8(0); i < num; i++ {
		bases = append(bases, g.Iterate())
	}

	g.Bases = bases
}
