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
	*events.OutgoingAgreementUnmarshaller
	committee.Committee
}

func NewAgreementSigner(committee committee.Committee, keys *user.Keys) *agreementSigner {
	return &agreementSigner{
		Committee:                     committee,
		eventSigner:                   newEventSigner(keys),
		OutgoingAgreementUnmarshaller: events.NewOutgoingAgreementUnmarshaller(),
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

/*
// CreateStepVotes creates the aggregated representation of the BLS signature and BLS PublicKeys used to sign the reduction
func CreateStepVotes(header *events.Header, evs []wire.Event, committee committee.Committee) (*events.StepVotes, error) {
	var apk *bls.Apk
	provisioners := sortedset.New()
	sigma := &bls.Signature{}

	for i, ev := range evs {
		rEv := ev.(*events.Reduction)
		provisioners.Insert(sender)

	}

	bitset := committee.Pack(provisioners, header.Round, header.Step)
	stepVotes := events.NewStepVotes(apk.Marshal(), bitset, sigma.Marshal())
	return stepVotes, nil
}
*/
