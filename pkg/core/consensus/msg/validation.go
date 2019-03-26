package msg

import (
	"bytes"
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

func VerifyEd25519Signature(messageBytes *bytes.Buffer) error {
	var signedMessage []byte
	if err := encoding.Read512(messageBytes, &signedMessage); err != nil {
		return err
	}

	var pubKey []byte
	if err := encoding.Read256(messageBytes, &pubKey); err != nil {
		return err
	}

	if !ed25519.Verify(pubKey, messageBytes.Bytes(), signedMessage) {
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
