package msg

import (
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/crypto/bls"
	"golang.org/x/crypto/ed25519"
)

func VerifyEd25519Signature(pubKey, message, signature []byte) error {
	if !ed25519.Verify(pubKey, message, signature) {
		return errors.New("ed25519 verification failed")
	}

	return nil
}

func VerifyBLSSignature(pubKeyBytes, message, signature []byte) error {
	pubKeyBLS := &bls.PublicKey{}
	if err := pubKeyBLS.Unmarshal(pubKeyBytes); err != nil {
		return err
	}

	sig := &bls.Signature{}
	if err := sig.Decompress(signature); err != nil {
		return err
	}

	apk := bls.NewApk(pubKeyBLS)
	return bls.Verify(apk, message, sig)
}
