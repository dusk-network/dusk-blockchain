package committee

import (
	"bytes"
	"encoding/hex"
	"sync"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
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
	}

	Store struct {
		publisher    wire.EventPublisher
		lock         sync.RWMutex
		provisioners *user.Provisioners
		// TODO: should this be round dependent?
		totalWeight uint64

		// subscriber channels
		newProvisionerChan    chan *provisioner
		removeProvisionerChan chan []byte
		// own keys (TODO: this should just be BLSPubKey)
		keys *user.Keys
	}
)

// LaunchCommitteeStore creates a component that listens to changes to the Provisioners
func LaunchCommitteeStore(eventBroker wire.EventBroker, keys *user.Keys) *Store {
	store := &Store{
		keys:         keys,
		publisher:    eventBroker,
		provisioners: &user.Provisioners{},
		// TODO: consider adding a consensus.Validator preprocessor
		newProvisionerChan:    initNewProvisionerCollector(eventBroker),
		removeProvisionerChan: InitRemoveProvisionerCollector(eventBroker),
	}
	go store.Listen()
	return store
}

func (c *Store) Listen() {
	for {
		select {
		case newProvisioner := <-c.newProvisionerChan:
			c.lock.Lock()
			if err := c.provisioners.AddMember(newProvisioner.pubKeyEd,
				newProvisioner.pubKeyBLS, newProvisioner.amount); err != nil {
				c.lock.Unlock()
				log.WithError(err).WithFields(log.Fields{
					"process": "committeeStore",
					"bls_key": newProvisioner.pubKeyBLS,
				}).Warnln("error in adding a provisioner member")
				continue
			}
			c.totalWeight += newProvisioner.amount
			c.lock.Unlock()
		case pubKeyBLS := <-c.removeProvisionerChan:
			stake, err := c.provisioners.GetStake(pubKeyBLS)
			if err != nil {
				panic(err)
			}

			c.lock.Lock()
			if err := c.provisioners.RemoveMember(pubKeyBLS); err != nil {
				c.lock.Unlock()
				log.WithError(err).WithFields(log.Fields{
					"process": "committeeStore",
					"bls_key": pubKeyBLS,
				}).Warnln("error in removing a provisioner member")
				continue
			}
			c.totalWeight -= stake
			c.lock.Unlock()
		}
	}
}

func (c *Store) AmMember(round uint64, step uint8) bool {
	return c.IsMember(c.keys.BLSPubKey.Marshal(), round, step)
}

// IsMember checks if the BLS key belongs to one of the Provisioners in the committee
func (c *Store) IsMember(pubKeyBLS []byte, round uint64, step uint8) bool {
	p := c.copyProvisioners()
	votingCommittee := p.CreateVotingCommittee(round, c.getTotalWeight(), step)
	return votingCommittee.IsMember(pubKeyBLS)
}

// Quorum returns the amount of votes to reach a quorum
func (c *Store) Quorum() int {
	p := c.copyProvisioners()
	committeeSize := p.VotingCommitteeSize()
	quorum := int(float64(committeeSize) * 0.75)
	return quorum
}

// ReportAbsentees will send public keys of absent provisioners to the moderator
func (c *Store) ReportAbsentees(evs []wire.Event, round uint64, step uint8) error {
	absentees := c.extractAbsentees(evs, round, step)
	for absentee := range absentees {
		absenteeBytes, err := hex.DecodeString(absentee)
		if err != nil {
			return err
		}
		c.publisher.Publish(msg.AbsenteesTopic, bytes.NewBuffer(absenteeBytes))
	}
	return nil
}

func (c *Store) extractAbsentees(evs []wire.Event, round uint64, step uint8) user.VotingCommittee {
	p := c.copyProvisioners()
	votingCommittee := p.CreateVotingCommittee(round, c.getTotalWeight(), step)
	for _, ev := range evs {
		senderStr := hex.EncodeToString(ev.Sender())
		delete(votingCommittee, senderStr)
	}
	return votingCommittee
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

func (c *Store) getTotalWeight() uint64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	totalWeight := c.totalWeight
	return totalWeight
}
