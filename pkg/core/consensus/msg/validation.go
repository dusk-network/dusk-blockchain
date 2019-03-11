package msg

import (
	"bytes"
	"errors"

	"github.com/cretz/bine/torutil/ed25519"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

func Validate(messageBytes *bytes.Buffer) error {
	signedMessage, err := decodeSignedMessage(messageBytes)
	if err != nil {
		return err
	}

	pubKey, err := decodePubKey(messageBytes)
	if err != nil {
		return err
	}

	if !ed25519.Verify(pubKey, messageBytes.Bytes(), signedMessage) {
		return errors.New("ed25519 verification failed")
	}

	return nil
}

func decodeSignedMessage(messageBytes *bytes.Buffer) ([]byte, error) {
	var signedMessage []byte
	if err := encoding.Read512(messageBytes, &signedMessage); err != nil {
		return nil, err
	}

	return signedMessage, nil
}

func decodePubKey(messageBytes *bytes.Buffer) ([]byte, error) {
	var pubKey []byte
	if err := encoding.Read256(messageBytes, &pubKey); err != nil {
		return nil, err
	}

	return pubKey, nil
}
