package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/notary"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"golang.org/x/crypto/ed25519"
)

type sigSetAgreementSigner struct {
	*eventSigner
	*notary.SigSetEventUnmarshaller
}

func newSigSetAgreementSigner(keys *user.Keys, c committee.Committee) *sigSetAgreementSigner {
	return &sigSetAgreementSigner{
		eventSigner:             newEventSigner(keys, c),
		SigSetEventUnmarshaller: notary.NewSigSetEventUnmarshaller(),
	}
}

func (as *sigSetAgreementSigner) eligibleToVote() bool {
	return as.committee.IsMember(as.Keys.BLSPubKey.Marshal())
}

func (as *sigSetAgreementSigner) addSignatures(ev wire.Event) (*bytes.Buffer, error) {
	e := ev.(*notary.SigSetEvent)
	if err := as.signBLS(e); err != nil {
		return nil, err
	}

	signBuffer := new(bytes.Buffer)
	if err := as.Marshal(signBuffer, e); err != nil {
		return nil, err
	}

	as.signEd25519(e, signBuffer.Bytes())
	buffer := new(bytes.Buffer)
	if err := as.Marshal(buffer, e); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (as *sigSetAgreementSigner) signBLS(ev wire.Event) error {
	e := ev.(*notary.SigSetEvent)
	buffer := new(bytes.Buffer)
	if err := as.MarshalVoteSet(buffer, e.VoteSet); err != nil {
		return err
	}

	signedVoteSet, err := bls.Sign(as.BLSSecretKey, as.BLSPubKey, buffer.Bytes())
	e.SignedVoteSet = signedVoteSet.Compress()
	e.EventHeader.PubKeyBLS = as.BLSPubKey.Marshal()
	return err
}

func (as *sigSetAgreementSigner) signEd25519(e *notary.SigSetEvent, eventBytes []byte) {
	signature := ed25519.Sign(*as.EdSecretKey, eventBytes)
	e.EventHeader.Signature = signature
	e.EventHeader.PubKeyEd = as.EdPubKeyBytes()
}
