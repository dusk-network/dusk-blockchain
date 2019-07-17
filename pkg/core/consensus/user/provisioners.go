package user

import (
	"fmt"
	"sync"
	"unsafe"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database/heavy"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
	"golang.org/x/crypto/ed25519"
)

type (
	// Member contains the bytes of a provisioner's Ed25519 public key,
	// the bytes of his BLS public key, and how much he has staked.
	Member struct {
		PublicKeyEd  ed25519.PublicKey
		PublicKeyBLS bls.PublicKey
		Stakes       []*stake
	}

	// Provisioners is a slice of Members, and makes up the current provisioner committee. It implements sort.Interface
	Provisioners struct {
		lock      sync.RWMutex
		set       sortedset.Set
		members   map[string]*Member
		sizeCache map[uint64]int
	}

	stake struct {
		amount      uint64
		startHeight uint64
		endHeight   uint64
	}
)

func (m *Member) addStake(stake *stake) {
	m.Stakes = append(m.Stakes, stake)
}

func (m *Member) removeStake(idx int) {
	m.Stakes[idx] = m.Stakes[len(m.Stakes)-1]
	m.Stakes = m.Stakes[:len(m.Stakes)-1]
}

// NewProvisioners returns an initialized Provisioners struct.
func NewProvisioners(db database.DB) (*Provisioners, uint64, error) {
	p := &Provisioners{
		set:       sortedset.New(),
		members:   make(map[string]*Member),
		sizeCache: make(map[uint64]int),
	}

	if db == nil {
		_, db = heavy.CreateDBConnection()
	}

	totalWeight := p.repopulate(db)
	return p, totalWeight, nil
}

func (p *Provisioners) repopulate(db database.DB) uint64 {
	var currentHeight uint64
	err := db.View(func(t database.Transaction) error {
		var err error
		currentHeight, err = t.FetchCurrentHeight()
		return err
	})

	if err != nil {
		currentHeight = 0
	}

	searchingHeight := uint64(0)
	if currentHeight > transactions.MaxLockTime {
		searchingHeight = currentHeight - transactions.MaxLockTime
	}

	var totalWeight uint64
	for {
		var blk *block.Block
		err := db.View(func(t database.Transaction) error {
			hash, err := t.FetchBlockHashByHeight(searchingHeight)
			if err != nil {
				return err
			}

			blk, err = t.FetchBlock(hash)
			return err
		})

		if err != nil {
			break
		}

		for _, tx := range blk.Txs {
			stake, ok := tx.(*transactions.Stake)
			if !ok {
				continue
			}

			p.AddMember(stake.PubKeyEd, stake.PubKeyBLS, stake.GetOutputAmount(), searchingHeight, searchingHeight+stake.Lock)
			totalWeight += stake.GetOutputAmount()
		}

		searchingHeight++
	}
	return totalWeight
}

// Size returns the amount of Members contained within a Provisioners struct.
func (p *Provisioners) Size(round uint64) int {
	p.lock.Lock()
	defer p.lock.Unlock()
	if size, ok := p.sizeCache[round]; ok {
		return size
	}

	// If we don't have the size for this round yet, we can delete the old ones.
	// It is safe to assume that we have gone to the next round, and we can free up that storage.
	p.sizeCache = make(map[uint64]int)
	size := p.membersAt(round)
	p.sizeCache[round] = size
	return size
}

// strPk is an efficient way to turn []byte into string
func strPk(pk []byte) string {
	return *(*string)(unsafe.Pointer(&pk))
}

// MemberAt returns the Member at a certain index.
func (p *Provisioners) MemberAt(i int) *Member {
	p.lock.RLock()
	defer p.lock.RUnlock()
	bigI := p.set[i]
	return p.members[strPk(bigI.Bytes())]
}

// Calculate how many members are active on a given round.
func (p *Provisioners) membersAt(round uint64) int {
	size := 0
	for _, member := range p.members {
		for _, stake := range member.Stakes {
			// If there is at least one stake active on or before this round, they are counted.
			if stake.startHeight <= round {
				size++
				break
			}
		}
	}

	return size
}

// GetMember returns a member of the provisioners from its BLS public key.
func (p *Provisioners) GetMember(pubKeyBLS []byte) *Member {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.members[strPk(pubKeyBLS)]
}

// AddMember will add a Member to the Provisioners by using the bytes of a BLS public key.
func (p *Provisioners) AddMember(pubKeyEd, pubKeyBLS []byte, amount, startHeight, endHeight uint64) error {
	if len(pubKeyEd) != 32 {
		return fmt.Errorf("public key is %v bytes long instead of 32", len(pubKeyEd))
	}

	if len(pubKeyBLS) != 129 {
		return fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	i := strPk(pubKeyBLS)
	stake := &stake{amount, startHeight, endHeight}

	p.lock.Lock()
	defer p.lock.Unlock()
	// Check for duplicates
	inserted := p.set.Insert(pubKeyBLS)
	if !inserted {
		// If they already exist, just add their new stake
		p.members[i].addStake(stake)
		return nil
	}

	// This is a new provisioner, so let's initialize the Member struct and add them to the list
	m := &Member{}
	m.PublicKeyEd = ed25519.PublicKey(pubKeyEd)

	pubKey := &bls.PublicKey{}
	if err := pubKey.Unmarshal(pubKeyBLS); err != nil {
		return err
	}

	m.PublicKeyBLS = *pubKey
	m.addStake(stake)

	p.members[i] = m
	return nil
}

// Remove a Member, designated by their BLS public key.
func (p *Provisioners) Remove(pubKeyBLS []byte) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.members, strPk(pubKeyBLS))

	// Reset size cache, as it might change after removing provisioners.
	p.sizeCache = make(map[uint64]int)
	return p.set.Remove(pubKeyBLS)
}

func (p *Provisioners) RemoveExpired(round uint64) uint64 {
	var totalRemoved uint64
	for pk, member := range p.members {
		for i := 0; i < len(member.Stakes); i++ {
			if member.Stakes[i].endHeight < round {
				totalRemoved += member.Stakes[i].amount
				member.removeStake(i)
				// If they have no stakes left, we should remove them entirely to keep our Size() calls accurate.
				if len(member.Stakes) == 0 {
					p.Remove([]byte(pk))
				}

				// Reset index
				i = -1
			}
		}
	}

	return totalRemoved
}

// GetStake will find a certain provisioner in the committee by BLS public key,
// and return their stake.
func (p *Provisioners) GetStake(pubKeyBLS []byte) (uint64, error) {
	if len(pubKeyBLS) != 129 {
		return 0, fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	p.lock.RLock()
	defer p.lock.RUnlock()

	i := strPk(pubKeyBLS)
	m, found := p.members[i]
	if !found {
		return 0, fmt.Errorf("public key %v not found among provisioner set", pubKeyBLS)
	}

	var totalStake uint64
	for _, stake := range m.Stakes {
		totalStake += stake.amount
	}

	return totalStake, nil
}
