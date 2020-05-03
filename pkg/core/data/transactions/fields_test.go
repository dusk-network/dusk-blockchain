package transactions

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	assert "github.com/stretchr/testify/require"
)

type noteTable struct {
	name string
	mock func() *rusk.Note
	test func(*Note)
}

func table(t *testing.T) (*assert.Assertions, []noteTable) {
	assert := assert.New(t)

	return assert, []noteTable{
		{
			"one",
			mockNote1,
			func(tNote *Note) {
				assert.Equal([]byte{0x55, 0x66}, tNote.ValueCommitment.Data)
				assert.Equal([]byte{0x55, 0x66}, tNote.TransparentBlindingFactor.Data)
				assert.Equal(uint64(122), tNote.TransparentValue)
			},
		},
		{
			"two",
			mockNote2,
			func(tNote *Note) {
				assert.Equal([]byte{0x56, 0x67}, tNote.EncryptedBlindingFactor)
				assert.Equal([]byte{0x12, 0x02}, tNote.EncryptedValue)
			},
		},
	}
}

func TestWireUnMarshalNote(t *testing.T) {
	assert, ttest := table(t)
	for _, tt := range ttest {
		_, n, err := notes(tt.mock)
		assert.NoError(err)

		marshaled := new(bytes.Buffer)
		assert.NoError(MarshalNote(marshaled, *n))

		tNote := new(Note)
		assert.NoError(UnmarshalNote(marshaled, tNote))
		tt.test(tNote)
	}
}

func TestRuskUnMarshalNote(t *testing.T) {
	assert := assert.New(t)
	note, n, err := notes(mockNote1)
	assert.NoError(err)

	outNote := new(rusk.Note)
	assert.NoError(MNote(outNote, n))
	assert.Equal(outNote.Nonce.Bs, note.Nonce.Bs)
	assert.Equal(outNote.RG.Y, note.RG.Y)
	assert.Equal(outNote.PkR.Y, note.PkR.Y)
	assert.Equal(outNote.ValueCommitment.Data, note.ValueCommitment.Data)
	assert.Equal(outNote.BlindingFactor, note.BlindingFactor)
	assert.Equal(outNote.Value, note.Value)
}

func TestInconsistentNote(t *testing.T) {
	assert := assert.New(t)
	_, _, err := notes(mockNote3)
	assert.Error(err)
}

func notes(mockNote func() *rusk.Note) (*rusk.Note, *Note, error) {
	note := mockNote()
	n := new(Note)
	err := UNote(note, n)
	return note, n, err
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
