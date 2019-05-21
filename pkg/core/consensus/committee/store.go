package committee

import (
	"bytes"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

const (
	ReductionCommitteeSize = 50
	AgreementCommitteeSize = 100
)

type (
	// Committee is the interface for operations depending on the set of Provisioners
	// extracted for a given step
	Committee interface {
		wire.EventPrioritizer

		AmMember(uint64, uint8) bool
		IsMember([]byte, uint64, uint8) bool
		Quorum() int
		ReportAbsentees([]wire.Event, uint64, uint8) error

		// Pack creates a uint64 bit representation of a sorted sub-committee for a
		// given round and step
		Pack(sortedset.Set, uint64, uint8) uint64

		// Unpack reconstruct a sorted sub-committee from a uint64 bitset, a round
		// and a step
		Unpack(uint64, uint64, uint8) sortedset.Set
	}

	// Store is the component that contains a set of provisioners, and provides
	// access to this set, allowing clients to obtain consensus-related information.
	// This struct is shared by Phase structs in the node.
	Store struct {
		publisher    wire.EventPublisher
		lock         sync.RWMutex
		provisioners *user.Provisioners
		// TODO: should this be round dependent?
		totalWeight uint64

		removeProvisionerChan chan []byte

		// own keys (TODO: this should just be BLSPubKey)
		keys *user.Keys
	}

	// Phase is a wrapper around the Store struct, and contains the phase-specific
	// information, as well as a voting committee cache.
	// TODO: better name
	Phase struct {
		*Store
		committeeSize  int
		round          uint64
		lock           sync.RWMutex
		committeeCache map[uint8]user.VotingCommittee
	}
)

// LaunchCommitteeStore creates a component that listens to changes to the Provisioners
func LaunchCommitteeStore(eventBroker wire.EventBroker, keys *user.Keys) *Store {
	store := &Store{
		keys:                  keys,
		publisher:             eventBroker,
		provisioners:          user.NewProvisioners(),
		removeProvisionerChan: InitRemoveProvisionerCollector(eventBroker),
	}
	eventBroker.SubscribeCallback(msg.NewProvisionerTopic, store.AddProvisioner)
	go store.Listen()
	return store
}

func NewCommittee(store *Store, size int) *Phase {
	return &Phase{
		Store:          store,
		committeeSize:  size,
		committeeCache: make(map[uint8]user.VotingCommittee),
	}
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

// AmMember checks if we are part of the committee.
func (p *Phase) AmMember(round uint64, step uint8) bool {
	return p.IsMember(p.Store.keys.BLSPubKeyBytes, round, step)
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (p *Phase) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	votingCommittee := p.upsertCommitteeCache(round, step)
	return votingCommittee.IsMember(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (p *Phase) Quorum() int {
	return int(float64(p.size()) * 0.75)
}

func (p *Phase) size() int {
	provisioners := p.copyProvisioners()
	if provisioners.Size() > p.committeeSize {
		return p.committeeSize
	}
	return provisioners.Size()
}

// Pack creates a uint64 bitset representation of a Committee subset for a given round and step
func (p *Phase) Pack(set sortedset.Set, round uint64, step uint8) uint64 {
	votingCommittee := p.upsertCommitteeCache(round, step)
	return votingCommittee.Bits(set)
}

// Unpack the Committee subset from a uint64 bitset representation for a give round and step
func (p *Phase) Unpack(bitset uint64, round uint64, step uint8) sortedset.Set {
	votingCommittee := p.upsertCommitteeCache(round, step)
	return votingCommittee.Intersect(bitset)
}

// ReportAbsentees will send public keys of absent provisioners to the moderator
func (p *Phase) ReportAbsentees(evs []wire.Event, round uint64, step uint8) error {
	absentees := p.extractAbsentees(evs, round, step)
	for _, absentee := range absentees.MemberKeys() {
		p.Store.publisher.Publish(msg.AbsenteesTopic, bytes.NewBuffer(absentee))
	}
	return nil
}

func (p *Phase) extractAbsentees(evs []wire.Event, round uint64, step uint8) user.VotingCommittee {
	votingCommittee := p.upsertCommitteeCache(round, step)
	for _, ev := range evs {
		votingCommittee.Remove(ev.Sender())
	}
	return votingCommittee
}

func (p *Phase) upsertCommitteeCache(round uint64, step uint8) user.VotingCommittee {
	if round > p.round {
		p.round = round
		p.lock.Lock()
		p.committeeCache = make(map[uint8]user.VotingCommittee)
		p.lock.Unlock()
	}
	p.lock.RLock()
	votingCommittee, found := p.committeeCache[step]
	p.lock.RUnlock()
	if !found {
		provisioners := p.copyProvisioners()
		votingCommittee = *provisioners.CreateVotingCommittee(round, p.getTotalWeight(),
			step, p.size())
		p.lock.Lock()
		p.committeeCache[step] = votingCommittee
		p.lock.Unlock()
	}
	return votingCommittee
}

// Priority returns false in case pubKey2 has higher stake than pubKey1
func (c *Store) Priority(ev1, ev2 wire.Event) bool {
	p := c.copyProvisioners()
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
