package notary

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
)

func TestSimpleBlockCollection(t *testing.T) {
	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(2)
	committeeMock.On("VerifyVoteSet",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything).Return(nil)
	committeeMock.On("IsMember", mock.AnythingOfType("[]uint8")).Return(true)

	blockChan := make(chan []byte, 1)
	validateFunc := func(*bytes.Buffer) error {
		return nil
	}

	bc := NewBlockCollector(committeeMock, blockChan, validateFunc)
	bc.UpdateRound(1)

	blockHash, err := crypto.RandEntropy(32)
	assert.Empty(t, err)

	ev1, err := mockCommitteeEventBuffer(blockHash, 1, 1)
	assert.Empty(t, err)
	ev2, err := mockCommitteeEventBuffer(blockHash, 1, 1)
	assert.Empty(t, err)
	ev3, err := mockCommitteeEventBuffer(blockHash, 1, 1)
	assert.Empty(t, err)

	bc.Collect(ev1)
	bc.Collect(ev2)
	bc.Collect(ev3)

	res := <-blockChan
	assert.Equal(t, blockHash, res)
}
