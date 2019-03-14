package user

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

const addProvisionerTopic = "addprovisioner"

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

// Threshold returns the voting threshold for a round
func (c CommitteeStore) Threshold() int {
	committeeSize := len(*c.provisioners)
	if committeeSize > 50 {
		committeeSize = 50
	}

	threshold := int(float64(committeeSize) * 0.75)
	return threshold
}

func (c CommitteeStore) VerifyVoteSet(voteSet []*msg.Vote, hash []byte, round uint64,
	step uint8) *prerror.PrError {

	for _, vote := range voteSet {
		if !fromValidStep(vote.Step, step) {
			return prerror.New(prerror.Low, errors.New("vote does not belong to vote set"))
		}

		votingCommittee, err := c.provisioners.CreateVotingCommittee(round,
			c.TotalWeight, vote.Step)
		if err != nil {
			return prerror.New(prerror.High, err)
		}

		if err := checkVoterEligibility(vote.PubKeyBLS, votingCommittee); err != nil {
			return err
		}

		if err := msg.VerifyBLSSignature(vote.PubKeyBLS, vote.VotedHash,
			vote.SignedHash); err != nil {

			return prerror.New(prerror.Low, errors.New("BLS verification failed"))
		}
	}

	return nil
}

func fromValidStep(voteStep, setStep uint8) bool {
	return voteStep == setStep || voteStep+1 == setStep
}

func checkVoterEligibility(pubKeyBLS []byte,
	votingCommittee map[string]uint8) *prerror.PrError {

	pubKeyStr := hex.EncodeToString(pubKeyBLS)
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
