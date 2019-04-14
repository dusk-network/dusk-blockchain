package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
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

func newAgreementSigner(keys *user.Keys, c committee.Committee) *agreementSigner {
	return &agreementSigner{
		eventSigner:           newEventSigner(keys, c),
		AgreementUnMarshaller: events.NewAgreementUnMarshaller(),
	}
}

func (as *agreementSigner) eligibleToVote() bool {
	return as.committee.IsMember(as.Keys.BLSPubKey.Marshal())
}

func (as *agreementSigner) addSignatures(ev wire.Event) (*bytes.Buffer, error) {
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

func (as *agreementSigner) signBLS(ev wire.Event) error {
	e := ev.(*events.Agreement)
	buffer := new(bytes.Buffer)
	if err := as.MarshalVoteSet(buffer, e.VoteSet); err != nil {
		return err
	}

	signedVoteSet, err := bls.Sign(as.BLSSecretKey, as.BLSPubKey, buffer.Bytes())
	e.SignedVoteSet = signedVoteSet.Compress()
	e.Header.PubKeyBLS = as.BLSPubKey.Marshal()
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
