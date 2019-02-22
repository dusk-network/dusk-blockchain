package user

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/ed25519"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/sortition"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/block"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/consensusmsg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Global consensus variables
var (
	maxMembers          = 200
	MaxSteps      uint8 = 50
	StepTime            = 1 * time.Second
	CandidateTime       = 1 * time.Second
	CommitteeSize uint8 = 50
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
	K          []byte        // secret
	Keys       *Keys
	Magic      protocol.Magic
	Multiplier uint8 // Time multiplier

	// Block generator values
	D          uint64 // bidWeight
	X, Y, Z, M []byte
	Q          uint64

	// Provisioner values
	/// General
	Weight      uint64             // Amount this node has staked
	W           uint64             // Total stake weight of the network
	Certificate *block.Certificate // Block certificate to be constructed during consensus

	/// Block fields
	CandidateBlock   *block.Block         // Winning block from candidate collection
	BlockHash        []byte               // Block hash currently being voted on by this node
	WinningBlockHash []byte               // Block hash that won the block phase
	BlockVotes       []*consensusmsg.Vote // Vote set for block set agreement phase

	/// Signature set fields
	SigSetVotes       []*consensusmsg.Vote // Vote set for signature set agreement phase
	SigSetHash        []byte               // Hash of the signature vote set being voted on
	WinningSigSetHash []byte               // Winning signature set hash

	/// Tracking fields
	Committee        [][]byte          // Lexicogaphically ordered provisioner public keys
	CurrentCommittee [][]byte          // Set of public keys of committee members for a current step
	NodeWeights      map[string]uint64 // Other nodes' Ed25519 public keys mapped to their stake weights
	NodeBLS          map[string][]byte // Other nodes' BLS public keys mapped to their Ed25519 public keys

	/// Message channels
	CandidateScoreChan  chan *payload.MsgConsensus
	CandidateChan       chan *payload.MsgConsensus
	BlockReductionChan  chan *payload.MsgConsensus
	BlockAgreementChan  chan *payload.MsgConsensus
	SigSetCandidateChan chan *payload.MsgConsensus
	SigSetReductionChan chan *payload.MsgConsensus
	SigSetAgreementChan chan *payload.MsgConsensus
	QuitChan            chan bool
	StopChan            chan bool

	// Message queue
	*Queue

	// General functions
	GetAllTXs   func() []*transactions.Stealth
	HashVotes   func([]*consensusmsg.Vote) ([]byte, error)
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
func NewContext(tau, d, totalWeight, round uint64, seed []byte, magic protocol.Magic, keys *Keys) (*Context, error) {
	if keys == nil {
		return nil, errors.New("context: keys is nil")
	}

	if keys.EdSecretKey == nil || keys.EdPubKey == nil || keys.BLSPubKey == nil || keys.BLSSecretKey == nil {
		return nil, errors.New("context: one of the keys to be used during the consensus is nil")
	}

	queue := &Queue{
		Map: new(sync.Map),
	}
	ctx := &Context{
		Version:             10000, // Placeholder
		Tau:                 tau,
		Threshold:           totalWeight / 5, // Placeholder
		Round:               round,
		Step:                1,
		Seed:                seed,
		Magic:               magic,
		Multiplier:          1,
		CandidateBlock:      &block.Block{},
		Certificate:         &block.Certificate{},
		D:                   d,
		CandidateScoreChan:  make(chan *payload.MsgConsensus, 100),
		CandidateChan:       make(chan *payload.MsgConsensus, 100),
		BlockReductionChan:  make(chan *payload.MsgConsensus, 100),
		BlockAgreementChan:  make(chan *payload.MsgConsensus, 100),
		SigSetCandidateChan: make(chan *payload.MsgConsensus, 100),
		SigSetReductionChan: make(chan *payload.MsgConsensus, 100),
		SigSetAgreementChan: make(chan *payload.MsgConsensus, 100),
		QuitChan:            make(chan bool, 1),
		StopChan:            make(chan bool, 1),
		Queue:               queue,
		W:                   totalWeight,
		GetAllTXs:           getAllTXs,
		HashVotes:           hashVotes,
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
	c.Multiplier = 1

	// Block generator
	c.K = nil
	c.X = nil
	c.Y = nil
	c.Z = nil
	c.M = nil
	c.Q = 0

	// Provisioner
	c.BlockHash = nil
	c.WinningBlockHash = nil
	c.BlockVotes = nil
	c.SigSetHash = nil
	c.WinningSigSetHash = nil
	c.SigSetVotes = nil
	c.CandidateBlock = &block.Block{}
	c.Step = 1
	c.Certificate = &block.Certificate{}
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

// CreateSignature will return the byte representation of a consensus message that
// is signed with Ed25519.
func (c *Context) CreateSignature(pl consensusmsg.Msg) ([]byte, error) {
	edMsg := make([]byte, 12)
	binary.LittleEndian.PutUint32(edMsg[0:], c.Version)
	binary.LittleEndian.PutUint64(edMsg[4:], c.Round)
	edMsg = append(edMsg, c.LastHeader.Hash...)
	edMsg = append(edMsg, byte(c.Step))
	edMsg = append(edMsg, byte(pl.Type()))
	buf := new(bytes.Buffer)
	if err := pl.Encode(buf); err != nil {
		return nil, err
	}

	edMsg = append(edMsg, buf.Bytes()...)
	return c.EDSign(c.Keys.EdSecretKey, edMsg), nil
}

// SetCommittee will set the committee for the current step
func (c *Context) SetCommittee() error {
	size := CommitteeSize
	if len(c.Committee) < int(CommitteeSize) {
		size = uint8(len(c.Committee))
	}

	currentCommittee, err := sortition.CreateCommittee(c.Round, c.W, c.Step, size,
		c.Committee, c.NodeWeights)
	if err != nil {
		return err
	}

	c.CurrentCommittee = currentCommittee
	return nil
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
	tx.TypeInfo = pl

	return []*transactions.Stealth{tx}
}

// HashVotes returns the hash of the passed vote set
func hashVotes(votes []*consensusmsg.Vote) ([]byte, error) {
	// Encode signature set
	buf := new(bytes.Buffer)
	for _, vote := range votes {
		if err := vote.Encode(buf); err != nil {
			return nil, err
		}
	}

	// Hash bytes and set it on context
	sigSetHash, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return sigSetHash, nil
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
	if err := bls.Verify(apk, msg, sig); err != nil {
		return err
	}

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
