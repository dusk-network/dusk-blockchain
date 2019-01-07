package consensus

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/ed25519"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/payload/transactions"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/protocol"
)

// Context will hold all the necessary functions and
// data types needed in the consensus
//XXX: depending on how we use context, we may want to unexport values and access them in a method via a mutex
type Context struct {
	Tau uint64
	k   []byte // secret

	// Consensus Values
	d          uint64 // bidWeight
	X, Y, Z, M []byte
	Q          uint64

	GetLastHeader func() *payload.BlockHeader
	GetAllTXs     func() []*transactions.Stealth

	BLSSign   func(key *bls.SecretKey, msg []byte) ([]byte, error)
	BLSVerify func(pkey *bls.PublicKey, msg []byte, signature []byte) error

	EDSign   func(privateKey *ed25519.PrivateKey, message []byte) ([]byte, error)
	EDVerify func(publicKey *ed25519.PublicKey, message, sig []byte) bool

	SendMessage func(magic protocol.Magic, p wire.Payload) error

	Keys *Keys
}

// NewContext will create a New context object
// with default or user-defined values
// XXX: Passing funcs as param will lead to big func sig. Put all funcs in a struct and pass struct in
// Check the func pointers are not nil, return err if so
func NewContext(Tau uint64, keys *Keys) (*Context, error) {

	if keys == nil {
		return nil, errors.New("Key is nil")
	}

	if keys.EdSecretKey == nil || keys.EdPubKey == nil || keys.BLSPubKey == nil || keys.BLSSecretKey == nil {
		return nil, errors.New("one of the keys to be used during the consensus is nil")
	}

	return &Context{
		Tau:           Tau,
		GetLastHeader: getLastHeader,
		GetAllTXs:     getAllTXs,
		BLSSign:       bLSSign,
		BLSVerify:     blsVerify,
		EDSign:        edSign,
		EDVerify:      edVerify,
		SendMessage:   send,
		Keys:          keys,
	}, nil
}

// Reset removes all information that was generated during the consensus
func (c *Context) Reset() {
	c.k = nil
	c.d = 0
	c.X = nil
	c.Y = nil
	c.Z = nil
	c.M = nil
	c.k = nil
	c.Q = 0
}

// Clear will remove all values created during consensus
// and all pointers to created during struct initialisation
func (c *Context) Clear() {
	c.Reset()
	c.EDSign = nil
	c.EDVerify = nil
	c.BLSSign = nil
	c.BLSVerify = nil
	c.GetAllTXs = nil
	c.GetLastHeader = nil
}

// dummy functions
func getLastHeader() *payload.BlockHeader {
	rand, _ := crypto.RandEntropy(32)
	return &payload.BlockHeader{
		Height:    100,
		Timestamp: 10,
		PrevBlock: rand,
		Seed:      rand,
		TxRoot:    rand,
		Hash:      rand,
		CertImage: rand,
	}
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
func blsVerify(pkey *bls.PublicKey, msg []byte, signature []byte) error {
	return nil
}

func edSign(privateKey *ed25519.PrivateKey, message []byte) ([]byte, error) {
	return crypto.RandEntropy(64)
}
func edVerify(publicKey *ed25519.PublicKey, message, sig []byte) bool {
	return true
}

func send(magic protocol.Magic, p wire.Payload) error {
	fmt.Printf("Sending message type %s\n", p.Command())
	return nil
}
