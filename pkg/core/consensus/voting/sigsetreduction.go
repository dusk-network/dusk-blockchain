package voting

import (
	"bytes"

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

func (ss *sigSetReductionSigner) addSignatures(ev wire.Event) (*bytes.Buffer, error) {
	e := ev.(*reduction.SigSetEvent)
	if err := ss.signBLS(e); err != nil {
		return nil, err
	}

	signBuffer := new(bytes.Buffer)
	if err := ss.Marshal(signBuffer, e); err != nil {
		return nil, err
	}

	ss.signEd25519(e, signBuffer.Bytes())
	buffer := new(bytes.Buffer)
	if err := ss.Marshal(buffer, e); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (ss *sigSetReductionSigner) signBLS(e *reduction.SigSetEvent) error {
	signedHash, err := bls.Sign(ss.BLSSecretKey, ss.BLSPubKey, e.VotedHash)
	e.SignedHash = signedHash.Compress()
	e.EventHeader.PubKeyBLS = ss.BLSPubKey.Marshal()
	return err
}

func (ss *sigSetReductionSigner) signEd25519(e *reduction.SigSetEvent, eventBytes []byte) {
	signature := ed25519.Sign(*ss.EdSecretKey, eventBytes)
	e.EventHeader.Signature = signature
	e.EventHeader.PubKeyEd = ss.EdPubKeyBytes()
}
