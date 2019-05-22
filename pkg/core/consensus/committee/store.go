package committee

import (
	"bytes"
	"sync"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
)

type (
	// Committee is the interface for operations depending on the set of Provisioners
	// extracted for a given step
	Committee interface {
		IsMember([]byte, uint64, uint8) bool
		Quorum() int
	}

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

		removeProvisionerChan chan []byte
	}

	// Extractor is a wrapper around the Store struct, and contains the phase-specific
	// information, as well as a voting committee cache. It calls methods on the
	// Store, passing it's own parameters to extract the desired info for a specific
	// phase.
	Extractor struct {
		*Store
		round          uint64
		lock           sync.RWMutex
		committeeCache map[uint8]user.VotingCommittee
	}
)

// LaunchCommitteeStore creates a component that listens to changes to the Provisioners
func LaunchCommitteeStore(eventBroker wire.EventBroker) *Store {
	store := &Store{
		provisioners:          user.NewProvisioners(),
		removeProvisionerChan: InitRemoveProvisionerCollector(eventBroker),
	}
	eventBroker.SubscribeCallback(msg.NewProvisionerTopic, store.AddProvisioner)
	go store.Listen()
	return store
}

func NewExtractor(eventBroker wire.EventBroker) *Extractor {
	return &Extractor{
		Store:          LaunchCommitteeStore(eventBroker),
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
	return nil
}

func (p *Extractor) UpsertCommitteeCache(round uint64, step uint8, size int) user.VotingCommittee {
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
		provisioners := p.Provisioners()
		votingCommittee = *provisioners.CreateVotingCommittee(round, p.getTotalWeight(),
			step, size)
		p.lock.Lock()
		p.committeeCache[step] = votingCommittee
		p.lock.Unlock()
	}
	return votingCommittee
}

func (c *Store) Provisioners() *user.Provisioners {
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
