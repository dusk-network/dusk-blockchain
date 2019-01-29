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
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

type role struct {
	part  string
	round uint64
	step  uint8
}

// Global consensus variables
var (
	maxMembers          = 200
	MaxSteps      uint8 = 50
	stepTime            = 20 * time.Second
	candidateTime       = 60 * time.Second
)

// Context will hold all the necessary functions and
// data types needed in the consensus
//XXX: depending on how we use context, we may want to unexport values and access them in a method via a mutex
type Context struct {
	// Common variables
	Version    uint32
	Tau        uint64
	Threshold  uint64
	Round      uint64        // Current round
	Step       uint8         // Current step
	Seed       []byte        // Round seed
	LastHeader *block.Header // Previous block
	k          []byte        // secret
	Keys       *Keys
	Magic      protocol.Magic

	// Block generator values
	d          uint64 // bidWeight
	X, Y, Z, M []byte
	Q          uint64

	// Provisioner values
	// General
	weight    uint64 // Amount this node has staked
	W         uint64 // Total stake weight of the network
	Score     []byte // Sortition score of this node
	votes     uint64 // Sortition votes of this node
	VoteLimit uint64 // Votes needed to decide on a block
	Empty     bool   // Whether or not the block being voted on is empty

	// Block fields
	CandidateBlock *block.Block         // Block kept from candidate collection
	BlockHash      []byte               // Block hash currently being voted on by this node
	BlockVotes     []*consensusmsg.Vote // Vote set for block set agreement phase
	BlockSetHash   []byte               // Hash of the block vote set being voted on

	// Signature set fields
	SigSetVotes []*consensusmsg.Vote // Vote set for signature set agreement phase
	SigSetHash  []byte               // Hash of the signature vote set being voted on

	// Tracking fields
	NodeWeights map[string]uint64 // Other nodes' Ed25519 public keys mapped to their stake weights
	NodeBLS     map[string][]byte // Other nodes' BLS public keys mapped to their Ed25519 public keys

	// Message channels
	CandidateScoreChan  chan *payload.MsgConsensus
	CandidateChan       chan *payload.MsgConsensus
	ReductionChan       chan *payload.MsgConsensus
	SetAgreementChan    chan *payload.MsgConsensus
	SigSetCandidateChan chan *payload.MsgConsensus
	SigSetVoteChan      chan *payload.MsgConsensus

	// General functions
	GetAllTXs   func() []*transactions.Stealth
	SendMessage func(magic protocol.Magic, p wire.Payload) error

	// Key functions
	BLSSign   func(sk *bls.SecretKey, pk *bls.PublicKey, msg []byte) ([]byte, error)
	BLSVerify func(pkey, msg, signature []byte) error
	EDSign    func(privateKey *ed25519.PrivateKey, message []byte) []byte
	EDVerify  func(publicKey, message, sig []byte) bool
}

// NewContext will create a New context object with default or user-defined values
// XXX: Passing funcs as param will lead to big func sig. Put all funcs in a struct and pass struct in
// Check the func pointers are not nil, return err if so
func NewContext(tau, totalWeight, round uint64, seed []byte, magic protocol.Magic, keys *Keys) (*Context, error) {
	if keys == nil {
		return nil, errors.New("context: keys is nil")
	}

	if keys.EdSecretKey == nil || keys.EdPubKey == nil || keys.BLSPubKey == nil || keys.BLSSecretKey == nil {
		return nil, errors.New("context: one of the keys to be used during the consensus is nil")
	}

	ctx := &Context{
		Version:             10000, // Placeholder
		Tau:                 tau,
		Threshold:           totalWeight / 5, // Placeholder
		Round:               round,
		Seed:                seed,
		Magic:               magic,
		CandidateScoreChan:  make(chan *payload.MsgConsensus, 100),
		CandidateChan:       make(chan *payload.MsgConsensus, 100),
		ReductionChan:       make(chan *payload.MsgConsensus, 100),
		SetAgreementChan:    make(chan *payload.MsgConsensus, 100),
		SigSetCandidateChan: make(chan *payload.MsgConsensus, 100),
		SigSetVoteChan:      make(chan *payload.MsgConsensus, 100),
		W:                   totalWeight,
		GetAllTXs:           getAllTXs,
		BLSSign:             bLSSign,
		BLSVerify:           blsVerify,
		EDSign:              edSign,
		EDVerify:            edVerify,
		SendMessage:         send,
		Keys:                keys,
		NodeWeights:         make(map[string]uint64),
		NodeBLS:             make(map[string][]byte),
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
	c.Step = 1
	c.CandidateBlock = nil
	c.BlockVotes = nil
	c.BlockSetHash = nil
	c.SigSetHash = nil
	c.SigSetVotes = nil
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
}

// dummy functions
func (c *Context) setLastHeader() *block.Header {
	rand, _ := crypto.RandEntropy(32)
	hdr := &block.Header{
		Height:    100,
		Timestamp: 10,
		PrevBlock: rand,
		Seed:      rand,
		TxRoot:    rand,
		Hash:      rand,
		CertHash:  rand,
	}

	c.LastHeader = hdr
	return hdr
}

func getAllTXs() []*transactions.Stealth {
	pl := transactions.NewStandard(100)
	tx := transactions.NewTX(transactions.StandardType, nil)

	keyImage, _ := crypto.RandEntropy(32)
	TxID, _ := crypto.RandEntropy(32)
	sig, _ := crypto.RandEntropy(32)

	input := transactions.NewInput(keyImage, TxID, 0, sig)
	pl.AddInput(input)

	dest, _ := crypto.RandEntropy(32)
	proof, _ := crypto.RandEntropy(32)
	output := transactions.NewOutput(100, dest, proof)
	pl.AddOutput(output)

	tx.R = sig

	return []*transactions.Stealth{tx}
}

// Sign with BLS and return the compressed signature
func bLSSign(sk *bls.SecretKey, pk *bls.PublicKey, msg []byte) ([]byte, error) {
	sig, err := bls.Sign(sk, pk, msg)
	if err != nil {
		return nil, err
	}

	return sig.Compress(), nil
}

// Verify a compressed bls signature
func blsVerify(pkey, msg, signature []byte) error {
	pk := &bls.PublicKey{}
	if err := pk.Unmarshal(pkey); err != nil {
		return err
	}

	sig := &bls.Signature{}
	if err := sig.Decompress(signature); err != nil {
		return err
	}

	apk := bls.NewApk(pk)
	return bls.Verify(apk, msg, sig)
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
