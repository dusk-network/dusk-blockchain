package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"golang.org/x/crypto/ed25519"
)

type agreementSigner struct {
	*eventSigner
	*events.AgreementUnMarshaller
}

func NewAgreementSigner(keys *user.Keys) *agreementSigner {
	return &agreementSigner{
		eventSigner:           newEventSigner(keys),
		AgreementUnMarshaller: events.NewAgreementUnMarshaller(),
	}
}

func (as *agreementSigner) AddSignatures(ev wire.Event) (*bytes.Buffer, error) {
	e := ev.(*events.Agreement)
	if err := as.signBLS(e); err != nil {
		return nil, err
	}

	buffer := new(bytes.Buffer)
	if err := as.Marshal(buffer, e); err != nil {
		return nil, err
	}

	message := as.signEd25519(e, buffer)
	return message, nil
}

func (as *agreementSigner) SignVotes(votes []wire.Event) ([]byte, error) {
	buffer := new(bytes.Buffer)
	if err := as.MarshalVoteSet(buffer, votes); err != nil {
		return nil, err
	}

	signedVoteSet, err := bls.Sign(as.BLSSecretKey, as.BLSPubKey, buffer.Bytes())
	if err != nil {
		return nil, err
	}
	return signedVoteSet.Compress(), nil
}

func (as *agreementSigner) signBLS(ev wire.Event) error {
	var err error
	e := ev.(*events.Agreement)
	e.Header.PubKeyBLS = as.BLSPubKey.Marshal()
	e.SignedVoteSet, err = as.SignVotes(e.VoteSet)
	if err != nil {
		return err
	}
	return err
}

func (as *agreementSigner) signEd25519(e *events.Agreement, eventBuf *bytes.Buffer) *bytes.Buffer {
	signature := ed25519.Sign(*as.EdSecretKey, eventBuf.Bytes())
	buf := new(bytes.Buffer)
	if err := encoding.Write512(buf, signature); err != nil {
		panic(err)
	}

	if err := encoding.Write256(buf, as.EdPubKeyBytes()); err != nil {
		panic(err)
	}

	if _, err := buf.Write(eventBuf.Bytes()); err != nil {
		panic(err)
	}

	return buf
}
