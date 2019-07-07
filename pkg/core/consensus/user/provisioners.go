package user

import (
	"fmt"
	"sync"
	"unsafe"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/database"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
	"golang.org/x/crypto/ed25519"
)

type (
	// Member contains the bytes of a provisioner's Ed25519 public key,
	// the bytes of his BLS public key, and how much he has staked.
	Member struct {
		PublicKeyEd  ed25519.PublicKey
		PublicKeyBLS bls.PublicKey
		Stake        uint64
		StartHeight  uint64
	}

	// Provisioners is a slice of Members, and makes up the current provisioner committee. It implements sort.Interface
	Provisioners struct {
		lock      sync.RWMutex
		set       sortedset.Set
		members   map[string]*Member
		sizeCache map[uint64]int
	}
)

// NewProvisioners returns an initialized Provisioners struct.
func NewProvisioners(db database.DB) *Provisioners {
	p := &Provisioners{
		set:       sortedset.New(),
		members:   make(map[string]*Member),
		sizeCache: make(map[uint64]int),
	}

	if db == nil {
		drvr, err := database.From(cfg.Get().Database.Driver)
		if err != nil {
			return nil, err
		}

		db, err = drvr.Open(cfg.Get().Database.Dir, protocol.MagicFromConfig(), false)
		if err != nil {
			return nil, err
		}
	}

	p.repopulate(db)
	return p
}

func (p *Provisioners) repopulate(db database.DB) {
	var currentHeight uint64
	err := db.View(func(t database.Transaction) error {
		state, err := t.FetchState()
		if err != nil {
			return err
		}

		header, err := t.FetchBlockHeader(state.TipHash)
		if err != nil {
			return err
		}

		currentHeight = header.Height
		return nil
	})

	searchingHeight := 0
	if currentHeight > transactions.MaxLockTime {
		searchingHeight = currentHeight - transactions.MaxLockTime
	}

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

			p.AddMember(stake.PubKeyEd, stake.PubKeyBLS, stake.GetOutputAmount(), stake.Lock)
		}

		searchingHeight++
	}
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
		if member.StartHeight <= round {
			size++
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

// AddMember will add a Member to the Provisioners by using the bytes of a BLS public key. Returns the index at which the key has been inserted. If the member alredy exists, AddMember overrides its stake with the new one
func (p *Provisioners) AddMember(pubKeyEd, pubKeyBLS []byte, stake, startHeight uint64) error {
	if len(pubKeyEd) != 32 {
		return fmt.Errorf("public key is %v bytes long instead of 32", len(pubKeyEd))
	}

	if len(pubKeyBLS) != 129 {
		return fmt.Errorf("public key is %v bytes long instead of 129", len(pubKeyBLS))
	}

	m := &Member{}
	m.PublicKeyEd = ed25519.PublicKey(pubKeyEd)

	pubKey := &bls.PublicKey{}
	if err := pubKey.Unmarshal(pubKeyBLS); err != nil {
		return err
	}

	m.PublicKeyBLS = *pubKey
	m.Stake = stake
	m.StartHeight = startHeight

	i := strPk(pubKeyBLS)

	p.lock.Lock()
	defer p.lock.Unlock()
	// Check for duplicates
	inserted := p.set.Insert(pubKeyBLS)
	if !inserted {
		p.members[i].Stake = stake
		return nil
	}

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

	return m.Stake, nil
}
