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

func (as *blockAgreementSigner) addBLSPubKey(ev wire.Event) {
	e := ev.(*notary.BlockEvent)
	e.EventHeader.PubKeyBLS = as.BLSPubKey.Marshal()
}

func (as *blockAgreementSigner) signBLS(ev wire.Event) error {
	e := ev.(*notary.BlockEvent)
	buffer := new(bytes.Buffer)
	if err := as.MarshalVoteSet(buffer, e.VoteSet); err != nil {
		return err
	}

	signedVoteSet, err := bls.Sign(as.BLSSecretKey, as.BLSPubKey, buffer.Bytes())
	e.SignedVoteSet = signedVoteSet.Compress()
	return err
}

func (as *blockAgreementSigner) signEd25519(marshalledEvent []byte) []byte {
	return ed25519.Sign(*as.EdSecretKey, marshalledEvent)
}
