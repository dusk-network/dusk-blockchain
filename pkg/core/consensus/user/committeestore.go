package user

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

const addProvisionerTopic = "addprovisioner"

type Committee interface {
	// isMember can accept a BLS Public Key or an Ed25519
	IsMember([]byte) bool
	GetVotingCommittee(uint64, uint8) (map[string]uint8, error)
	VerifyVoteSet(voteSet []*msg.Vote, hash []byte, round uint64, step uint8) *prerror.PrError
	Quorum() int
}

type CommitteeStore struct {
	eventBus              *wire.EventBus
	addProvisionerChannel <-chan *bytes.Buffer
	addProvisionerID      uint32

	provisioners *Provisioners
	TotalWeight  uint64
}

func NewCommitteeStore(eventBus *wire.EventBus) *CommitteeStore {
	addProvisionerChannel := make(chan *bytes.Buffer, 100)

	committeeStore := &CommitteeStore{
		eventBus:              eventBus,
		addProvisionerChannel: addProvisionerChannel,
		provisioners:          &Provisioners{},
	}

	addProvisionerID := committeeStore.eventBus.Subscribe(addProvisionerTopic,
		addProvisionerChannel)
	committeeStore.addProvisionerID = addProvisionerID

	return committeeStore
}

func (c *CommitteeStore) Listen() {
	for {
		select {
		case newProvisionerBytes := <-c.addProvisionerChannel:
			pubKeyEd, pubKeyBLS, amount, err := decodeNewProvisioner(newProvisionerBytes)
			if err != nil {
				// Log
				return
			}

			if err := c.provisioners.AddMember(pubKeyEd, pubKeyBLS, amount); err != nil {
				break
			}

			c.TotalWeight += amount
			c.eventBus.Publish(msg.ProvisionerAddedTopic, newProvisionerBytes)
		}
	}
}

// Get the provisioner committee and return it
func (c CommitteeStore) Get() Provisioners {
	return *c.provisioners
}

// PartakesInCommittee checks if the BLS key belongs to one of the Provisioners in the committee
func (c *CommitteeStore) IsMember(pubKeyBLS []byte) bool {
	return c.provisioners.GetMember(pubKeyBLS) != nil
}

func (c *CommitteeStore) GetVotingCommittee(round uint64, step uint8) (map[string]uint8, error) {
	return c.provisioners.CreateVotingCommittee(round, c.TotalWeight, step)
}

// Quorum returns the amount of votes to reach a quorum
func (c CommitteeStore) Quorum() int {
	committeeSize := len(*c.provisioners)
	if committeeSize > 50 {
		committeeSize = 50
	}

	quorum := int(float64(committeeSize) * 0.75)
	return quorum
}

func (c CommitteeStore) VerifyVoteSet(voteSet []*msg.Vote, hash []byte, round uint64,
	step uint8) *prerror.PrError {

	var amountOfVotes uint8

	for _, vote := range voteSet {
		if err := checkDuplicates(voteSet, vote); err != nil {
			return err
		}

		if !fromValidStep(vote.Step, step) {
			return prerror.New(prerror.Low, errors.New("vote does not belong to vote set"))
		}

		votingCommittee, err := c.provisioners.CreateVotingCommittee(round,
			c.TotalWeight, vote.Step)
		if err != nil {
			return prerror.New(prerror.High, err)
		}

		pubKeyStr := hex.EncodeToString(vote.PubKeyBLS)
		if err := checkVoterEligibility(pubKeyStr, votingCommittee); err != nil {
			return err
		}

		if err := msg.VerifyBLSSignature(vote.PubKeyBLS, vote.VotedHash,
			vote.SignedHash); err != nil {

			return prerror.New(prerror.Low, errors.New("BLS verification failed"))
		}

		amountOfVotes += votingCommittee[pubKeyStr]
	}

	if int(amountOfVotes) < c.Quorum() {
		return prerror.New(prerror.Low, errors.New("vote set too small"))
	}

	return nil
}

func checkDuplicates(voteSet []*msg.Vote, vote *msg.Vote) *prerror.PrError {
	for _, v := range voteSet {
		if v.Equals(vote) {
			return prerror.New(prerror.Low, errors.New("vote set contains duplicate vote"))
		}
	}

	return nil
}

func fromValidStep(voteStep, setStep uint8) bool {
	return voteStep == setStep || voteStep+1 == setStep
}

func checkVoterEligibility(pubKeyStr string,
	votingCommittee map[string]uint8) *prerror.PrError {

	if votingCommittee[pubKeyStr] == 0 {
		return prerror.New(prerror.Low, errors.New("voter is not eligible to vote"))
	}

	return nil
}

func decodeNewProvisioner(r io.Reader) ([]byte, []byte, uint64, error) {
	var pubKeyEd []byte
	if err := encoding.Read256(r, &pubKeyEd); err != nil {
		return nil, nil, 0, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
		return nil, nil, 0, err
	}

	var amount uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &amount); err != nil {
		return nil, nil, 0, err
	}

	return pubKeyEd, pubKeyBLS, amount, nil
}
