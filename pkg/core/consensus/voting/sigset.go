package voting

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"golang.org/x/crypto/ed25519"
)

type sigSetSigner struct {
	*eventSigner
	*committee.NotaryEventUnMarshaller
}

func newSigSetSigner(keys *user.Keys, c committee.Committee) *sigSetSigner {
	return &sigSetSigner{
		eventSigner:             newEventSigner(keys, c),
		NotaryEventUnMarshaller: committee.NewNotaryEventUnMarshaller(func([]byte, []byte, []byte) error { return nil }),
	}
}

func (ss *sigSetSigner) eligibleToVote() bool {
	return ss.committee.IsMember(ss.Keys.BLSPubKey.Marshal())
}

func (ss *sigSetSigner) addSignatures(ev wire.Event) (*bytes.Buffer, error) {
	e := ev.(*committee.NotaryEvent)
	e.PubKeyEd = nil
	e.Signature = nil
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

func (ss *sigSetSigner) signEd25519(e *committee.NotaryEvent, eventBytes []byte) {
	signature := ed25519.Sign(*ss.EdSecretKey, eventBytes)
	e.EventHeader.Signature = signature
	e.EventHeader.PubKeyEd = ss.EdPubKeyBytes()
}
