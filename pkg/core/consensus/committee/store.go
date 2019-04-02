package committee

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

// Store is the component that handles Committee formation and management
type Store struct {
	eventBus              *wire.EventBus
	addProvisionerChannel <-chan *bytes.Buffer
	addProvisionerID      uint32

	provisioners *user.Provisioners
	TotalWeight  uint64
}

// NewCommitteeStore creates a new Store
func NewCommitteeStore(eventBus *wire.EventBus) *Store {
	addProvisionerChannel := make(chan *bytes.Buffer, 100)

	committeeStore := &Store{
		eventBus:              eventBus,
		addProvisionerChannel: addProvisionerChannel,
		provisioners:          &user.Provisioners{},
	}

	addProvisionerID := committeeStore.eventBus.Subscribe(msg.NewProvisionerTopic,
		addProvisionerChannel)
	committeeStore.addProvisionerID = addProvisionerID

	return committeeStore
}

// Listen for
func (c *Store) Listen() {
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
func (c Store) Get() user.Provisioners {
	return *c.provisioners
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (c *Store) IsMember(pubKey []byte) bool {
	if len(pubKey) == 129 {
		return c.provisioners.GetMemberBLS(pubKey) != nil
	}

	if len(pubKey) == 32 {
		return c.provisioners.GetMemberEd(pubKey) != nil
	}

	return false
}

// GetVotingCommittee returns a voting comittee
func (c *Store) GetVotingCommittee(round uint64, step uint8) (map[string]uint8, error) {
	return c.provisioners.CreateVotingCommittee(round, c.TotalWeight, step)
}

// Quorum returns the amount of votes to reach a quorum
func (c Store) Quorum() int {
	committeeSize := len(*c.provisioners)
	if committeeSize > 50 {
		committeeSize = 50
	}

	quorum := int(float64(committeeSize) * 0.75)
	return quorum
}

// Priority returns true in case pubKey2 has higher stake than pubKey1
func (c Store) Priority(ev1, ev2 wire.Event) wire.Event {
	if _, ok := ev1.(*NotaryEvent); !ok {
		return ev2
	}

	m1 := c.provisioners.GetMemberBLS(ev1.Sender())
	m2 := c.provisioners.GetMemberBLS(ev2.Sender())
	if m1 == m2 {
		return ev1
	}

	if m1 == nil {
		return ev2
	}

	if m2 == nil {
		return ev1
	}

	if m1.Stake < m2.Stake {
		return ev2
	}
	return ev1
}

// VerifyVoteSet checks the signature of the set
func (c Store) VerifyVoteSet(voteSet []wire.Event, hash []byte, round uint64,
	step uint8) *prerror.PrError {

	// var amountOfVotes uint8

	// for _, vote := range voteSet {
	// if !fromValidStep(vote.Step, step) {
	// 	return prerror.New(prerror.Low, errors.New("vote does not belong to vote set"))
	// }

	// votingCommittee, err := c.provisioners.CreateVotingCommittee(round,
	// 	c.TotalWeight, vote.Step)
	// if err != nil {
	// 	return prerror.New(prerror.High, err)
	// }

	// pubKeyStr := hex.EncodeToString(vote.Sender())
	// if err := checkVoterEligibility(pubKeyStr, votingCommittee); err != nil {
	// 	return err
	// }

	// if err := msg.VerifyBLSSignature(vote.PubKeyBLS, vote.VotedHash,
	// 	vote.SignedHash); err != nil {

	// 	return prerror.New(prerror.Low, errors.New("BLS verification failed"))
	// }

	// amountOfVotes += votingCommittee[pubKeyStr]
	// }

	// if int(amountOfVotes) < c.Quorum() {
	// 	return prerror.New(prerror.Low, errors.New("vote set too small"))
	// }

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
