package key

import (
	"bytes"
	"io"

	"github.com/dusk-network/dusk-crypto/bls"
	"golang.org/x/crypto/ed25519"
)

// ConsensusKeys are the keys used during consensus
type ConsensusKeys struct {
	BLSPubKey      *bls.PublicKey
	BLSPubKeyBytes []byte
	BLSSecretKey   *bls.SecretKey
	EdPubKey       *ed25519.PublicKey
	EdPubKeyBytes  []byte
	EdSecretKey    *ed25519.PrivateKey
}

// NewRandConsensusKeys will generate and return new bls and ed25519
// keys to be used in consensus
func NewRandConsensusKeys() (ConsensusKeys, error) {
	var keys ConsensusKeys
	var err error
	for {
		keys, err = NewConsensusKeysFromReader(nil)
		if err == io.EOF {
			continue
		}
		break
	}

	return keys, err
}

// NewConsensusKeysFromReader takes a reader and uses it as a seed for generating random ConsensusKeys
func NewConsensusKeysFromReader(r io.Reader) (ConsensusKeys, error) {
	blsPub, blsPriv, err := bls.GenKeyPair(r)
	if err != nil {
		return ConsensusKeys{}, err
	}

	edPub, edPriv, err := ed25519.GenerateKey(r)
	if err != nil {
		return ConsensusKeys{}, err
	}

	return ConsensusKeys{
		BLSPubKey:      blsPub,
		BLSSecretKey:   blsPriv,
		EdPubKey:       &edPub,
		EdPubKeyBytes:  []byte(edPub),
		EdSecretKey:    &edPriv,
		BLSPubKeyBytes: blsPub.Marshal(),
	}, nil
}

// NewConsensusKeysFromBytes gets an array of bytes as seed for the pseudo-random generation of ConsensusKeys
func NewConsensusKeysFromBytes(seed []byte) (ConsensusKeys, error) {

	r := bytes.NewBuffer(seed)
	r.Grow(len(seed))
	blsPub, blsPriv, err := bls.GenKeyPair(r)
	if err != nil {
		return ConsensusKeys{}, err
	}

	r = bytes.NewBuffer(seed)
	r.Grow(len(seed))
	edPub, edPriv, err := ed25519.GenerateKey(r)
	if err != nil {
		return ConsensusKeys{}, err
	}

	return ConsensusKeys{
		BLSPubKey:      blsPub,
		BLSSecretKey:   blsPriv,
		EdPubKey:       &edPub,
		EdPubKeyBytes:  []byte(edPub),
		EdSecretKey:    &edPriv,
		BLSPubKeyBytes: blsPub.Marshal(),
	}, nil
}
