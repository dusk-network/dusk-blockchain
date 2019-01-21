package consensus

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/ed25519"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Global consensus variables
var (
	maxMembers       = 200
	maxSteps   uint8 = 50
	stepTime         = 20 * time.Second
)

// Context will hold all the necessary functions and
// data types needed in the consensus
//XXX: depending on how we use context, we may want to unexport values and access them in a method via a mutex
type Context struct {
	// Common variables
	Tau        uint64
	Round      uint64               // Current round
	step       uint8                // Current step
	Seed       []byte               // Round seed
	LastHeader *payload.BlockHeader // Previous block
	k          []byte               // secret
	Keys       *Keys
	Magic      protocol.Magic

	// Block generator values
	d          uint64 // bidWeight
	X, Y, Z, M []byte
	Q          uint64

	// Provisioner values
	weight    uint64 // Amount this node has staked
	W         uint64 // Total stake weight of the network
	Score     []byte // Sortition score of this node
	votes     int    // Sortition votes of this node
	VoteLimit uint64 // Votes needed to decide on a block
	Empty     bool   // Whether or not the block being voted on is empty
	BlockHash []byte // Block hash currently being voted on by this node
	// q          chan bool               // Channel used to halt BA phase in case of a decision from set agreement
	// Committee  *Committee              // Provisioner committee
	// Signatures []*payload.SignatureSet // Result of set agreement phase

	// General functions
	GetAllTXs   func() []*transactions.Stealth
	SendMessage func(magic protocol.Magic, p wire.Payload) error

	// Key functions
	BLSSign   func(key *bls.SecretKey, msg []byte) ([]byte, error)
	BLSVerify func(pkey, msg []byte, signature []byte) error
	EDSign    func(privateKey *ed25519.PrivateKey, message []byte) []byte
	EDVerify  func(publicKey, message, sig []byte) bool
}

// NewGeneratorContext will create a New context object for block generators
// with default or user-defined values
// XXX: Passing funcs as param will lead to big func sig. Put all funcs in a struct and pass struct in
// Check the func pointers are not nil, return err if so
func NewGeneratorContext(Tau uint64, keys *Keys) (*Context, error) {

	if keys == nil {
		return nil, errors.New("Key is nil")
	}

	if keys.EdSecretKey == nil || keys.EdPubKey == nil || keys.BLSPubKey == nil || keys.BLSSecretKey == nil {
		return nil, errors.New("one of the keys to be used during the consensus is nil")
	}

	ctx := &Context{
		Tau:         Tau,
		GetAllTXs:   getAllTXs,
		BLSSign:     bLSSign,
		BLSVerify:   blsVerify,
		EDSign:      edSign,
		EDVerify:    edVerify,
		SendMessage: send,
		Keys:        keys,
	}

	ctx.setLastHeader()
	return ctx, nil
}

// NewProvisionerContext will create a New context object for provisioners
// with default or user-defined values
// XXX: Passing funcs as param will lead to big func sig. Put all funcs in a struct and pass struct in
// Check the func pointers are not nil, return err if so
func NewProvisionerContext(totalWeight, round uint64, seed []byte, magic protocol.Magic, keys *Keys) (*Context, error) {

	if keys == nil {
		return nil, errors.New("Key is nil")
	}

	if keys.EdSecretKey == nil || keys.EdPubKey == nil || keys.BLSPubKey == nil || keys.BLSSecretKey == nil {
		return nil, errors.New("one of the keys to be used during the consensus is nil")
	}

	ctx := &Context{
		Tau:         totalWeight / 5, // Placeholder
		Round:       round,
		Seed:        seed,
		Magic:       magic,
		W:           totalWeight,
		GetAllTXs:   getAllTXs,
		BLSSign:     bLSSign,
		BLSVerify:   blsVerify,
		EDSign:      edSign,
		EDVerify:    edVerify,
		SendMessage: send,
		Keys:        keys,
		// q:           make(chan bool),
		// Committee:   NewCommittee(),
	}

	ctx.setLastHeader()
	return ctx, nil
}

// Reset removes all information that was generated during the consensus
func (c *Context) Reset() {
	// Block generator
	c.k = nil
	c.d = 0
	c.X = nil
	c.Y = nil
	c.Z = nil
	c.M = nil
	c.k = nil
	c.Q = 0

	// Provisioner
	c.Score = nil
	c.votes = 0
	c.BlockHash = nil
	c.Empty = false
	c.step = 1
	// c.Committee.Clear()
}

// Clear will remove all values created during consensus
// and all pointers to created during struct initialisation
func (c *Context) Clear() {
	c.Reset()
	c.Seed = nil
	c.EDSign = nil
	c.EDVerify = nil
	c.BLSSign = nil
	c.BLSVerify = nil
	c.GetAllTXs = nil
	c.Keys = nil
	// c.Committee = nil
}

// RaiseVoteLimit will adjust the provisioner vote limit for the final step.
// func (c *Context) RaiseVoteLimit() {
// 	c.VoteLimit = uint8(float64(len(c.Committee.Members)) * 0.8)
// }

// SetCommittee will set up the provisioner committee for the current round.
// The function assumes that the array passed has been pre-sorted according to
// stake size. It will also set the provisioner vote limit.
// func (c *Context) SetCommittee(stakers [][]byte) {
// 	l := maxMembers
// 	if len(stakers) < maxMembers {
// 		l = len(stakers)
// 	}

// 	for i := 0; i < l; i++ {
// 		// score := DeterministicSortition(staker[i])
// 		// c.Committee.AddMember(staker[i], score)
// 	}

// 	// Set votelimit
// 	c.VoteLimit = uint8(float64(len(c.Committee.Members)) * 0.75)
// }

// dummy functions
func (c *Context) setLastHeader() *payload.BlockHeader {
	rand, _ := crypto.RandEntropy(32)
	hdr := &payload.BlockHeader{
		Height:    100,
		Timestamp: 10,
		PrevBlock: rand,
		Seed:      rand,
		TxRoot:    rand,
		Hash:      rand,
		CertImage: rand,
	}

	c.LastHeader = hdr
	return hdr
}

func getAllTXs() []*transactions.Stealth {
	tx := transactions.NewTX()

	keyImage, _ := crypto.RandEntropy(32)
	TxID, _ := crypto.RandEntropy(32)
	sig, _ := crypto.RandEntropy(32)

	input := transactions.NewInput(keyImage, TxID, 0, sig)
	tx.AddInput(input)

	dest, _ := crypto.RandEntropy(32)
	proof, _ := crypto.RandEntropy(32)
	output := transactions.NewOutput(100, dest, proof)
	tx.AddOutput(output)

	tx.AddTxPubKey(sig)

	return []*transactions.Stealth{tx}
}

// Use this until bls is implemented
func bLSSign(key *bls.SecretKey, msg []byte) ([]byte, error) {
	return crypto.RandEntropy(32)
}
func blsVerify(pkey, msg []byte, signature []byte) error {
	return nil
}

func edSign(privateKey *ed25519.PrivateKey, message []byte) []byte {
	return ed25519.Sign(*privateKey, message)
}
func edVerify(publicKey, message, sig []byte) bool {
	pk := ed25519.PublicKey(publicKey)
	return ed25519.Verify(pk, message, sig)
}

func send(magic protocol.Magic, p wire.Payload) error {
	fmt.Printf("Sending message type %s\n", p.Command())
	return nil
}
