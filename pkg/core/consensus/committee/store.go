package committee

import (
	"bytes"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

// Store is the component that handles Committee formation and management
type (
	// Committee is the interface for operations depending on the set of Provisioners extracted for a fiven step
	Committee interface {
		wire.EventPrioritizer
		// isMember can accept a BLS Public Key or an Ed25519
		AmMember(uint64, uint8) bool
		IsMember([]byte, uint64, uint8) bool
		Quorum() int
		ReportAbsentees([]wire.Event, uint64, uint8) error

		// Pack creates a uint64 bit representation of a sorted sub-committee for a given round and step
		Pack(sortedset.Set, uint64, uint8) uint64

		// Unpack reconstruct a sorted sub-committee from a uint64 bitset, a round and a step
		Unpack(uint64, uint64, uint8) sortedset.Set
	}

	Store struct {
		publisher    wire.EventPublisher
		lock         sync.RWMutex
		provisioners *user.Provisioners
		// TODO: should this be round dependent?
		totalWeight uint64
		round       uint64

		// subscriber channels
		removeProvisionerChan chan []byte
		// own keys (TODO: this should just be BLSPubKey)
		keys *user.Keys

		committeeCache map[uint8]user.VotingCommittee
	}
)

// LaunchCommitteeStore creates a component that listens to changes to the Provisioners
func LaunchCommitteeStore(eventBroker wire.EventBroker, keys *user.Keys) *Store {
	store := &Store{
		keys:         keys,
		publisher:    eventBroker,
		provisioners: user.NewProvisioners(),
		// TODO: consider adding a consensus.Validator preprocessor
		removeProvisionerChan: InitRemoveProvisionerCollector(eventBroker),
		committeeCache:        make(map[uint8]user.VotingCommittee),
	}
	eventBroker.SubscribeCallback(msg.NewProvisionerTopic, store.AddProvisioner)
	go store.Listen()
	return store
}

func (c *Store) Listen() {
	for {
		select {
		case pubKeyBLS := <-c.removeProvisionerChan:
			stake, err := c.provisioners.GetStake(pubKeyBLS)
			if err != nil {
				panic(err)
			}

			c.lock.Lock()
			c.provisioners.Remove(pubKeyBLS)
			c.totalWeight -= stake
			c.lock.Unlock()
		}
	}
}

func (c *Store) AddProvisioner(m *bytes.Buffer) error {
	newProvisioner, err := decodeNewProvisioner(m)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.provisioners.AddMember(newProvisioner.pubKeyEd,
		newProvisioner.pubKeyBLS, newProvisioner.amount); err != nil {
		return err
	}

	c.totalWeight += newProvisioner.amount
	c.publisher.Publish(msg.ProvisionerAddedTopic, bytes.NewBuffer(newProvisioner.pubKeyBLS))
	return nil
}

func (c *Store) AmMember(round uint64, step uint8) bool {
	return c.IsMember(c.keys.BLSPubKey.Marshal(), round, step)
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (c *Store) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	votingCommittee := c.upsertCommitteeCache(round, step)
	return votingCommittee.IsMember(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (c *Store) Quorum() int {
	p := c.copyProvisioners()
	committeeSize := p.VotingCommitteeSize()
	quorum := int(float64(committeeSize) * 0.75)
	return quorum
}

// Pack creates a uint64 bitset representation of a Committee subset for a given round and step
func (c *Store) Pack(set sortedset.Set, round uint64, step uint8) uint64 {
	p := c.copyProvisioners()
	votingCommittee := p.CreateVotingCommittee(round, c.getTotalWeight(), step)
	return votingCommittee.Bits(set)
}

// Unpack the Committee subset from a uint64 bitset representation for a give round and step
func (c *Store) Unpack(bitset uint64, round uint64, step uint8) sortedset.Set {
	p := c.copyProvisioners()
	votingCommittee := p.CreateVotingCommittee(round, c.getTotalWeight(), step)
	return votingCommittee.Intersect(bitset)
}

// ReportAbsentees will send public keys of absent provisioners to the moderator
func (c *Store) ReportAbsentees(evs []wire.Event, round uint64, step uint8) error {
	absentees := c.extractAbsentees(evs, round, step)
	for _, absentee := range absentees.MemberKeys() {
		c.publisher.Publish(msg.AbsenteesTopic, bytes.NewBuffer(absentee))
	}
	return nil
}

func (c *Store) extractAbsentees(evs []wire.Event, round uint64, step uint8) user.VotingCommittee {
	votingCommittee := c.upsertCommitteeCache(round, step)
	for _, ev := range evs {
		votingCommittee.Remove(ev.Sender())
	}
	return votingCommittee
}

func (c *Store) upsertCommitteeCache(round uint64, step uint8) user.VotingCommittee {
	if round > c.round {
		c.round = round
		c.lock.Lock()
		c.committeeCache = make(map[uint8]user.VotingCommittee)
		c.lock.Unlock()
	}
	c.lock.RLock()
	votingCommittee, found := c.committeeCache[step]
	c.lock.RUnlock()
	if !found {
		p := c.copyProvisioners()
		votingCommittee = *p.CreateVotingCommittee(round, c.getTotalWeight(), step)
		c.lock.Lock()
		c.committeeCache[step] = votingCommittee
		c.lock.Unlock()
	}
	return votingCommittee
}

// Priority returns false in case pubKey2 has higher stake than pubKey1
func (c *Store) Priority(ev1, ev2 wire.Event) bool {
	p := c.copyProvisioners()
	if _, ok := ev1.(*events.Agreement); !ok {
		return false
	}

	m1 := p.GetMember(ev1.Sender())
	m2 := p.GetMember(ev2.Sender())

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

func (c *Store) getTotalWeight() uint64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	totalWeight := c.totalWeight
	return totalWeight
}
