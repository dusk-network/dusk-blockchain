package transactions

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	"github.com/stretchr/testify/assert"
)

var table = []struct {
	name string
	mock func() *rusk.Note
	test func(*testing.T, *Note)
}{
	{
		"one",
		mockNote1,
		func(t *testing.T, tNote *Note) {
			if !assert.Equal(t, []byte{0x55, 0x66}, tNote.ValueCommitment.Data) {
				t.Fatal()
			}

			if !assert.Equal(t, []byte{0x55, 0x66}, tNote.BlindingFactor.TransparentBlindingFactor.Data) {
				t.Fatal()
			}

			if !assert.Equal(t, uint64(122), tNote.Value.TransparentValue) {
				t.Fatal()
			}
		},
	},
	{
		"two",
		mockNote2,
		func(t *testing.T, tNote *Note) {
			if !assert.Equal(t, []byte{0x56, 0x67}, tNote.BlindingFactor.EncryptedBlindingFactor) {
				t.Fatal()
			}

			if !assert.Equal(t, []byte{0x12, 0x02}, tNote.Value.EncryptedValue) {
				t.Fatal()
			}
		},
	},
	{
		"three",
		mockNote3,
		func(t *testing.T, tNote *Note) {
			if !assert.Nil(t, tNote.BlindingFactor.TransparentBlindingFactor) {
				t.Fatal()
			}

			if !assert.Equal(t, uint64(0), tNote.Value.TransparentValue) {
				t.Fatal()
			}
		},
	},
}

func TestNote(t *testing.T) {

	for _, tt := range table {
		note := tt.mock()
		b, _ := json.Marshal(note)
		n := new(Note)
		assert.NoError(t, json.Unmarshal(b, n))

		marshaled := new(bytes.Buffer)
		assert.NoError(t, MarshalNote(marshaled, *n))

		tNote := &Note{}
		assert.NoError(t, UnmarshalNote(marshaled, tNote))
		tt.test(t, tNote)
	}
}

func mockNote1() *rusk.Note {
	return &rusk.Note{
		NoteType:        0,
		Nonce:           &rusk.Nonce{Bs: []byte{0x11, 0x22}},
		RG:              &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
		PkR:             &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
		ValueCommitment: &rusk.Scalar{Data: []byte{0x55, 0x66}},
		BlindingFactor: &rusk.Note_TransparentBlindingFactor{
			TransparentBlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
		},
		Value: &rusk.Note_TransparentValue{
			TransparentValue: uint64(122),
		},
	}
}

func mockNote2() *rusk.Note {
	return &rusk.Note{
		NoteType:        1,
		Nonce:           &rusk.Nonce{},
		RG:              &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
		PkR:             &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
		ValueCommitment: &rusk.Scalar{Data: []byte{0x55, 0x66}},
		BlindingFactor: &rusk.Note_EncryptedBlindingFactor{
			EncryptedBlindingFactor: []byte{0x56, 0x67},
		},
		Value: &rusk.Note_EncryptedValue{
			EncryptedValue: []byte{0x12, 0x02},
		},
	}
}

func mockNote3() *rusk.Note {
	return &rusk.Note{
		NoteType:        1,
		Nonce:           &rusk.Nonce{Bs: []byte{0x11, 0x22}},
		RG:              &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
		PkR:             &rusk.CompressedPoint{Y: []byte{0x33, 0x44}},
		ValueCommitment: &rusk.Scalar{Data: []byte{0x55, 0x66}},
		BlindingFactor: &rusk.Note_TransparentBlindingFactor{
			TransparentBlindingFactor: &rusk.Scalar{Data: []byte{0x55, 0x66}},
		},
		Value: &rusk.Note_TransparentValue{
			TransparentValue: uint64(122),
		},
	}
}
