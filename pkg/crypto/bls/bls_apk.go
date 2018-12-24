package bls

import (
	"bytes"
	"math/big"

	"github.com/cloudflare/bn256"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/pkg/errors"
	"golang.org/x/crypto/sha3"
)

// This package implements the modified version of BLS which provides security vs rogue-key attack. For more information please consult https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html

// H1 is the hash function used to provide 128-bit security and to protect vs rogue-key attack
// TODO: consider using Blake2
var H1 = sha3.New256

// AggregateApk aggregates signatures and public keys into the σ ← σ1^t1 ⋯ σn^t ∈ G1
func AggregateApk(pks []*PublicKey, signatures []*Sig) (*Sig, error) {
	var apkSig *bn256.G1

	for i, pk := range pks {
		sigI := signatures[i].e
		h1 := H1()
		pkb, err := pk.MarshalBinary()

		if err != nil {
			return nil, err
		}

		t, err := hash.PerformHash(h1, pkb)
		if err != nil {
			return nil, err
		}

		tInt := new(big.Int).SetBytes(t)
		sigTI := sigI.ScalarBaseMult(tInt)

		if i == 0 {
			apkSig = sigTI
		} else {
			apkSig.Add(apkSig, sigTI)
		}
	}

	return &Sig{apkSig}, nil
}

// VerifyApk is the verification step of an aggregated apk signature
// According to Boneh, Drijvers and Neven paper on the rogue-key resilient BLS aggregation, a "designated combiner who computes the final signature" is appointed to create the signature.
// This is not exactly a centralized process, but still in a decentralized environment we need to see how to select this combiner
// See https://eprint.iacr.org/2018/483.pdf
func VerifyApk(pks []*PublicKey, msg []byte, signature *Sig) error {
	h0m, err := hashToPoint(msg)
	if err != nil {
		return err
	}

	var apkPk *bn256.G2

	for i := range pks {
		pkGxI := pks[i].gx
		h1 := H1()
		pkb, err := pks[i].MarshalBinary()

		if err != nil {
			return err
		}

		t, err := hash.PerformHash(h1, pkb)
		tInt := new(big.Int).SetBytes(t)

		if i == 0 {
			apkPk = pkGxI.ScalarBaseMult(tInt)
		} else {
			apkPk.Add(apkPk, pkGxI.ScalarBaseMult(tInt))
		}
	}

	pairApkH0m := bn256.Pair(h0m, apkPk).Marshal()
	pairG1Sig := bn256.Pair(signature.e, g2Base).Marshal()

	if !bytes.Equal(pairApkH0m, pairG1Sig) {
		return errors.New("bls apk: Invalid Signature")
	}

	return nil
}
