package transactions

import (
	"bytes"
	"testing"

	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
	assert "github.com/stretchr/testify/require"
)

type noteTable struct {
	name string //nolint
	mock *rusk.Note
	test func(*Note)
}

func table(t *testing.T) (*assert.Assertions, []noteTable) {
	assert := assert.New(t)

	return assert, []noteTable{
		{
			"one",
			RuskTransparentNote(),
			func(tNote *Note) {
				assert.Equal([]byte{0x55, 0x66}, tNote.ValueCommitment.Data)
				assert.Equal([]byte{0x55, 0x66}, tNote.TransparentBlindingFactor.Data)
				assert.Equal(uint64(122), tNote.TransparentValue)
			},
		},
		{
			"two",
			RuskObfuscatedNote(),
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
		n, err := notes(tt.mock)
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
	note := RuskTransparentNote()
	n, err := notes(note)
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
	_, err := notes(RuskInvalidNote())
	assert.Error(err)
}

func notes(note *rusk.Note) (*Note, error) {
	n := new(Note)
	err := UNote(note, n)
	return n, err
}
