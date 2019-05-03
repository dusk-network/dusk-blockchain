package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

type reductionSigner struct {
	*eventSigner
	*events.OutgoingReductionUnmarshaller
}

func NewReductionSigner(keys *user.Keys) *reductionSigner {
	return &reductionSigner{
		eventSigner:                   newEventSigner(keys),
		OutgoingReductionUnmarshaller: events.NewOutgoingReductionUnmarshaller(),
	}
}

func (bs *reductionSigner) Sign(buf *bytes.Buffer) error {
	e := events.NewReduction()
	if err := bs.Unmarshal(buf, e); err != nil {
		return err
	}

	if err := bs.signBLS(e); err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	if err := bs.Marshal(buffer, e); err != nil {
		return err
	}

	message := bs.signEd25519(e, buffer)
	*buf = *message
	return nil
}

func (bs *reductionSigner) signBLS(e *events.Reduction) error {
	signedHash, err := bls.Sign(bs.BLSSecretKey, bs.BLSPubKey, e.VotedHash)
	e.SignedHash = signedHash.Compress()
	e.Header.PubKeyBLS = bs.BLSPubKey.Marshal()
	return err
}

func (bs *reductionSigner) signEd25519(e *events.Reduction, eventBuf *bytes.Buffer) *bytes.Buffer {
	signature := ed25519.Sign(*bs.EdSecretKey, eventBuf.Bytes())
	buf := new(bytes.Buffer)
	if err := encoding.Write512(buf, signature); err != nil {
		panic(err)
	}

	if err := encoding.Write256(buf, bs.EdPubKeyBytes()); err != nil {
		panic(err)
	}

	if _, err := buf.Write(eventBuf.Bytes()); err != nil {
		panic(err)
	}

	return buf
}
