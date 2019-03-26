package voting

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"golang.org/x/crypto/ed25519"
)

type blockReductionSigner struct {
	*eventSigner
	*committee.ReductionEventUnMarshaller
}

func newBlockReductionSigner(keys *user.Keys, c committee.Committee) *blockReductionSigner {
	return &blockReductionSigner{
		eventSigner:                newEventSigner(keys, c),
		ReductionEventUnMarshaller: committee.NewReductionEventUnMarshaller(),
	}
}

func (bs *blockReductionSigner) eligibleToVote() bool {
	return bs.committee.IsMember(bs.Keys.BLSPubKey.Marshal())
}

func (bs *blockReductionSigner) addBLSPubKey(ev wire.Event) {
	e := ev.(*committee.ReductionEvent)
	e.EventHeader.PubKeyBLS = bs.BLSPubKey.Marshal()
}

func (bs *blockReductionSigner) signBLS(ev wire.Event) error {
	e := ev.(*committee.ReductionEvent)
	signedHash, err := bls.Sign(bs.BLSSecretKey, bs.BLSPubKey, e.VotedHash)
	e.SignedHash = signedHash.Compress()
	return err
}

func (bs *blockReductionSigner) signEd25519(marshalledEvent []byte) []byte {
	return ed25519.Sign(*bs.EdSecretKey, marshalledEvent)
}
