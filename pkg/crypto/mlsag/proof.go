package mlsag

import (
	"errors"
	"math/rand"
	"time"

	ristretto "github.com/bwesterb/go-ristretto"
)

type Proof struct {
	// index indicating the column of the secret keys
	index int

	// private keys corresponding to a column
	// in the matrix
	privKeys PrivKeys

	//Signer pubkeys
	signerPubKeys PubKeys

	// All pubKeys including the decoys
	// There should exist an index j such that
	// pubKeys[j][i] = privKeys[i] * G
	pubKeysMatrix []PubKeys

	// message to be signed
	msg []byte
}

func (p *Proof) addPubKeys(keys PubKeys) {
	//	// xxx: return an error if there is already a key vector in marix and their sizes do not match
	p.pubKeysMatrix = append(p.pubKeysMatrix, keys)
}

func (p *Proof) AddDecoy(keys PubKeys) {
	keys.decoy = true
	p.addPubKeys(keys)
}

func (p *Proof) AddDecoys(keys []PubKeys) {
	for _, key := range keys {
		p.AddDecoy(key)
	}
}

func (proof *Proof) addSignerPubKey() {
	// Add signers pubkey to matrix
	proof.signerPubKeys.decoy = false
	proof.addPubKeys(proof.signerPubKeys)
}

func (p *Proof) AddSecret(privKey ristretto.Scalar) {

	// Generate pubkey for given privkey
	rawPubKey := privKeyToPubKey(privKey)

	// Add pubkey to signers set of pubkeys
	p.signerPubKeys.AddPubKey(rawPubKey)
	// Add privkey to signers set of priv keys
	p.privKeys.AddPrivateKey(privKey)
}

func privKeyToPubKey(privkey ristretto.Scalar) ristretto.Point {
	var pubkey ristretto.Point
	pubkey.ScalarMultBase(&privkey)
	return pubkey
}

// shuffle all pubkeys and sets the index
func (p *Proof) shuffleSet() error {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := len(p.pubKeysMatrix) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		p.pubKeysMatrix[i], p.pubKeysMatrix[j] = p.pubKeysMatrix[j], p.pubKeysMatrix[i]
	}
	// XXX: Optimise away the below for loop by storing the index when appended
	// and following it in the first loop. We can also get rid of the decoy flag too

	// Find our index
	for i := range p.pubKeysMatrix {
		pubKey := p.pubKeysMatrix[i]
		if !pubKey.decoy {
			p.index = i
			return nil
		}
	}

	// If we get here, then we could not find the index of the signers pubkey
	return errors.New("could not find the index of the non-decoy vector of pubkeys")
}

func (p *Proof) LenMembers() int {
	return len(p.pubKeysMatrix)
}
