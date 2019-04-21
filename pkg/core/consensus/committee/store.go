package committee

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Store is the component that handles Committee formation and management
type (
	// Committee is the interface for operations depending on the set of Provisioners extracted for a fiven step
	Committee interface {
		wire.EventPrioritizer
		// isMember can accept a BLS Public Key or an Ed25519
		IsMember([]byte, uint64, uint8) bool
		Quorum() int
	}

	Store struct {
		eventBus *wire.EventBus

		lock         sync.RWMutex
		provisioners *user.Provisioners
		// TODO: should this be round dependent?
		TotalWeight uint64
	}
)

// LaunchCommitteeStore creates a component that listens to changes to the Provisioners
func LaunchCommitteeStore(eventBus *wire.EventBus) *Store {
	store := &Store{
		eventBus:     eventBus,
		provisioners: &user.Provisioners{},
	}
	listener := wire.NewTopicListener(eventBus, store, msg.NewProvisionerTopic)
	// TODO: consider adding a consensus.Validator preprocessor
	go listener.Accept()
	return store
}

func (c *Store) Collect(newProvisionerBytes *bytes.Buffer) error {
	var cpy bytes.Buffer
	r := io.TeeReader(newProvisionerBytes, &cpy)
	pubKeyEd, pubKeyBLS, amount, err := decodeNewProvisioner(r)
	if err != nil {
		// Log
		return err
	}

	c.lock.Lock()
	if err := c.provisioners.AddMember(pubKeyEd, pubKeyBLS, amount); err != nil {
		return err
	}
	c.lock.Unlock()

	c.TotalWeight += amount
	c.eventBus.Publish(msg.ProvisionerAddedTopic, &cpy)
	return nil
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (c *Store) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {

	p := c.copyProvisioners()
	votingCommittee := p.CreateVotingCommittee(round, c.TotalWeight, step)
	return votingCommittee.Has(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (c *Store) Quorum() int {
	p := c.copyProvisioners()
	committeeSize := p.VotingCommitteeSize()
	quorum := int(float64(committeeSize) * 0.75)
	return quorum
}

// Priority returns false in case pubKey2 has higher stake than pubKey1
func (c *Store) Priority(ev1, ev2 wire.Event) bool {
	p := c.copyProvisioners()
	if _, ok := ev1.(*events.Agreement); !ok {
		return false
	}

	m1 := p.GetMemberBLS(ev1.Sender())
	m2 := p.GetMemberBLS(ev2.Sender())

	if m1 == nil {
		return false
	}

	if m2 == nil {
		return true
	}

	return m1.Stake >= m2.Stake
}

func (c *Store) copyProvisioners() *user.Provisioners {
	c.lock.RLock()
	defer c.lock.RUnlock()
	provisioners := c.provisioners
	return provisioners
}

func decodeNewProvisioner(r io.Reader) ([]byte, []byte, uint64, error) {
	pubKeyEd := make([]byte, 0)
	if err := encoding.Read256(r, &pubKeyEd); err != nil {
		return nil, nil, 0, err
	}

	pubKeyBLS := make([]byte, 0)
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
		return nil, nil, 0, err
	}

	var amount uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &amount); err != nil {
		return nil, nil, 0, err
	}

	return pubKeyEd, pubKeyBLS, amount, nil
}
