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

func (as *sigSetAgreementSigner) signEd25519(marshalledEvent []byte) []byte {
	return ed25519.Sign(*as.EdSecretKey, marshalledEvent)
}
