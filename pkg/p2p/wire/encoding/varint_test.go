// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package encoding

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompactSize(t *testing.T) {
	a := uint64(1)
	b := uint64(1<<16 - 1)
	c := uint64(1<<32 - 1)
	d := uint64(1<<64 - 1)

	// Serialize
	buf := new(bytes.Buffer)
	if err := WriteVarInt(buf, a); err != nil {
		t.Fatal(err)
	}
	if err := WriteVarInt(buf, b); err != nil {
		t.Fatal(err)
	}
	if err := WriteVarInt(buf, c); err != nil {
		t.Fatal(err)
	}
	if err := WriteVarInt(buf, d); err != nil {
		t.Fatal(err)
	}

	// Deserialize
	e, err := ReadVarInt(buf)
	if err != nil {
		t.Fatal(err)
	}
	f, err := ReadVarInt(buf)
	if err != nil {
		t.Fatal(err)
	}
	g, err := ReadVarInt(buf)
	if err != nil {
		t.Fatal(err)
	}
	h, err := ReadVarInt(buf)
	if err != nil {
		t.Fatal(err)
	}

	// Compare
	assert.Equal(t, a, e)
	assert.Equal(t, b, f)
	assert.Equal(t, c, g)
	assert.Equal(t, d, h)
}
