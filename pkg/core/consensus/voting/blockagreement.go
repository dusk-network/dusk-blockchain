package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/notary"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"golang.org/x/crypto/ed25519"
)

type blockAgreementSigner struct {
	*eventSigner
	*committee.NotaryEventUnMarshaller
}

func newBlockAgreementSigner(keys *user.Keys, c committee.Committee) *blockAgreementSigner {
	return &blockAgreementSigner{
		eventSigner: newEventSigner(keys, c),
		NotaryEventUnMarshaller: committee.NewNotaryEventUnMarshaller(committee.NewReductionEventUnMarshaller(nil),
			msg.VerifyEd25519Signature),
	}
}

func (as *blockAgreementSigner) eligibleToVote() bool {
	return as.committee.IsMember(as.Keys.BLSPubKey.Marshal())
}

func (as *blockAgreementSigner) addSignatures(ev wire.Event) (*bytes.Buffer, error) {
	e := ev.(*notary.BlockEvent)
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

func (as *blockAgreementSigner) signBLS(ev wire.Event) error {
	e := ev.(*notary.BlockEvent)
	buffer := new(bytes.Buffer)
	if err := as.MarshalVoteSet(buffer, e.VoteSet); err != nil {
		return err
	}

	signedVoteSet, err := bls.Sign(as.BLSSecretKey, as.BLSPubKey, buffer.Bytes())
	e.SignedVoteSet = signedVoteSet.Compress()
	e.EventHeader.PubKeyBLS = as.BLSPubKey.Marshal()
	return err
}

func (as *blockAgreementSigner) signEd25519(e *notary.BlockEvent, eventBytes []byte) {
	signature := ed25519.Sign(*as.EdSecretKey, eventBytes)
	e.EventHeader.Signature = signature
	e.EventHeader.PubKeyEd = as.EdPubKeyBytes()
}
