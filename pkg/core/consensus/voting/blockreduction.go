package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
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
		ReductionEventUnMarshaller: committee.NewReductionEventUnMarshaller(msg.VerifyEd25519Signature),
	}
}

func (bs *blockReductionSigner) eligibleToVote() bool {
	return bs.committee.IsMember(bs.Keys.BLSPubKey.Marshal())
}

func (bs *blockReductionSigner) addSignatures(ev wire.Event) (*bytes.Buffer, error) {
	e := ev.(*committee.ReductionEvent)
	if err := bs.signBLS(e); err != nil {
		return nil, err
	}

	signBuffer := new(bytes.Buffer)
	if err := bs.Marshal(signBuffer, e); err != nil {
		return nil, err
	}

	bs.signEd25519(e, signBuffer.Bytes())
	buffer := new(bytes.Buffer)
	if err := bs.Marshal(buffer, e); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (bs *blockReductionSigner) signBLS(e *committee.ReductionEvent) error {
	signedHash, err := bls.Sign(bs.BLSSecretKey, bs.BLSPubKey, e.VotedHash)
	e.SignedHash = signedHash.Compress()
	e.EventHeader.PubKeyBLS = bs.BLSPubKey.Marshal()
	return err
}

func (bs *blockReductionSigner) signEd25519(e *committee.ReductionEvent, eventBytes []byte) {
	signature := ed25519.Sign(*bs.EdSecretKey, eventBytes)
	e.EventHeader.Signature = signature
	e.EventHeader.PubKeyEd = bs.EdPubKeyBytes()
}
