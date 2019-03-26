package voting

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"golang.org/x/crypto/ed25519"
)

type sigSetReductionSigner struct {
	*eventSigner
	*reduction.SigSetUnmarshaller
}

func newSigSetReductionSigner(keys *user.Keys, c committee.Committee) *sigSetReductionSigner {
	return &sigSetReductionSigner{
		eventSigner:        newEventSigner(keys, c),
		SigSetUnmarshaller: reduction.NewSigSetUnMarshaller(msg.VerifyEd25519Signature),
	}
}

func (ss *sigSetReductionSigner) eligibleToVote() bool {
	return ss.committee.IsMember(ss.Keys.BLSPubKey.Marshal())
}

func (ss *sigSetReductionSigner) addBLSPubKey(ev wire.Event) {
	e := ev.(*reduction.SigSetEvent)
	e.EventHeader.PubKeyBLS = ss.BLSPubKey.Marshal()
}

func (ss *sigSetReductionSigner) signBLS(ev wire.Event) error {
	e := ev.(*reduction.SigSetEvent)
	signedHash, err := bls.Sign(ss.BLSSecretKey, ss.BLSPubKey, e.VotedHash)
	e.SignedHash = signedHash.Compress()
	return err
}

func (ss *sigSetReductionSigner) signEd25519(marshalledEvent []byte) []byte {
	return ed25519.Sign(*ss.EdSecretKey, marshalledEvent)
}
