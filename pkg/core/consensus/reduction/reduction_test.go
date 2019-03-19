package reduction

import (
	"github.com/stretchr/testify/mock"
	"gitlab.dusk.network/dusk-core/dusk-go/mocks"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
)

func mockCommittee(quorum int, isMember bool, verification error,
	votingCommittee map[string]uint8) committee.Committee {

	committeeMock := &mocks.Committee{}
	committeeMock.On("Quorum").Return(quorum)
	committeeMock.On("VerifyVoteSet",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything).Return(verification)
	committeeMock.On("IsMember", mock.AnythingOfType("[]uint8")).Return(isMember)
	committeeMock.On("GetVotingCommittee", mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint8")).Return(votingCommittee, nil)
	return committeeMock
}
