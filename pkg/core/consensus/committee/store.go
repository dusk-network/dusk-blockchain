package committee

import (
	"bytes"
	"encoding/binary"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

type (
	// Committee is the interface for operations depending on the set of Provisioners
	// extracted for a given step
	Committee interface {
		IsMember([]byte, uint64, uint8) bool
		Quorum(uint64) int
	}

	// Foldable represents a Committee which can be packed into a bitset, to drastically
	// decrease the size needed for committee representation over the wire.
	Foldable interface {
		Committee
		Pack(sortedset.Set, uint64, uint8) uint64
		Unpack(uint64, uint64, uint8) sortedset.Set
	}

	// Store is the component that contains a set of provisioners, and provides
	// access to this set, allowing clients to obtain consensus-related information.
	// This struct is shared by Extractor structs in the node.
	Store struct {
		lock         sync.RWMutex
		provisioners *user.Provisioners
		// TODO: should this be round dependent?
		totalWeight uint64
	}

	// Extractor is a wrapper around the Store struct, and contains the phase-specific
	// information, as well as a voting committee cache. It calls methods on the
	// Store, passing its own parameters to extract the desired info for a specific
	// phase.
	Extractor struct {
		*Store
		round          uint64
		lock           sync.RWMutex
		committeeCache map[uint8]user.VotingCommittee
	}
)

// launchStore creates a component that listens to changes to the Provisioners
func launchStore(eventBroker wire.EventBroker, db database.DB) *Store {
	p, totalWeight, err := user.NewProvisioners(db)
	if err != nil {
		// If we can not repopulate our committee, we can not properly verify blocks
		// or run consensus. Thus, it's best to panic.
		panic(err)
	}

	store := &Store{
		provisioners: p,
		totalWeight:  totalWeight,
	}
	eventBroker.SubscribeCallback(msg.NewProvisionerTopic, store.AddProvisioner)
	eventBroker.SubscribeCallback(msg.RemoveProvisionerTopic, store.RemoveProvisioner)
	eventBroker.SubscribeCallback(msg.RoundUpdateTopic, store.RemoveExpiredProvisioners)
	return store
}

// NewExtractor returns a committee extractor which maintains its own store and cache.
func NewExtractor(eventBroker wire.EventBroker, db database.DB) *Extractor {
	return &Extractor{
		Store:          launchStore(eventBroker, db),
		committeeCache: make(map[uint8]user.VotingCommittee),
	}
}

// RemoveProvisioner will remove a provisioner from the Store's Provisioners objects.
func (s *Store) RemoveProvisioner(m *bytes.Buffer) error {
	stake, err := s.provisioners.GetStake(m.Bytes())
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.provisioners.Remove(m.Bytes())
	s.totalWeight -= stake
	return nil
}

func (s *Store) RemoveExpiredProvisioners(m *bytes.Buffer) error {
	round := binary.LittleEndian.Uint64(m.Bytes())
	s.lock.Lock()
	defer s.lock.Unlock()
	removedStakesAmount := s.provisioners.RemoveExpired(round)
	s.totalWeight -= removedStakesAmount
	return nil
}

// AddProvisioner will add a provisioner to the Store's Provisioners object.
func (s *Store) AddProvisioner(m *bytes.Buffer) error {
	newProvisioner, err := decodeNewProvisioner(m)
	if err != nil {
		return err
	}

	if err := s.provisioners.AddMember(newProvisioner.pubKeyEd,
		newProvisioner.pubKeyBLS, newProvisioner.amount, newProvisioner.startHeight, newProvisioner.endHeight); err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	s.totalWeight += newProvisioner.amount
	return nil
}

// UpsertCommitteeCache will return a voting committee for a given round, step and size.
// If the committee has not yet been produced before, it is put on the cache. If it has,
// it is simply retrieved and returned.
func (e *Extractor) UpsertCommitteeCache(round uint64, step uint8, size int) user.VotingCommittee {
	e.lock.Lock()
	defer e.lock.Unlock()
	if round > e.round {
		e.round = round
		e.committeeCache = make(map[uint8]user.VotingCommittee)
	}
	votingCommittee, found := e.committeeCache[step]
	if !found {
		provisioners := e.Provisioners()
		votingCommittee = *provisioners.CreateVotingCommittee(round, e.getTotalWeight(),
			step, size)
		e.committeeCache[step] = votingCommittee
	}
	return votingCommittee
}

// Provisioners returns a copy of the user.Provisioners object maintained by the Store.
func (s *Store) Provisioners() *user.Provisioners {
	s.lock.RLock()
	defer s.lock.RUnlock()
	provisioners := s.provisioners
	return provisioners
}

func (s *Store) getTotalWeight() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	totalWeight := s.totalWeight
	return totalWeight
}
