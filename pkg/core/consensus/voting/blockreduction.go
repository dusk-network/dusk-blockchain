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

type blockReductionSigner struct {
	*eventSigner
	*events.ReductionUnMarshaller
}

func newBlockReductionSigner(keys *user.Keys, c committee.Committee) *blockReductionSigner {
	return &blockReductionSigner{
		eventSigner:           newEventSigner(keys, c),
		ReductionUnMarshaller: events.NewReductionUnMarshaller(),
	}
}

func (bs *blockReductionSigner) eligibleToVote() bool {
	return bs.committee.IsMember(bs.Keys.BLSPubKey.Marshal())
}

func (bs *blockReductionSigner) addSignatures(ev wire.Event) (*bytes.Buffer, error) {
	e := ev.(*events.Reduction)
	if err := bs.signBLS(e); err != nil {
		return nil, err
	}

	buffer := new(bytes.Buffer)
	if err := bs.Marshal(buffer, e); err != nil {
		return nil, err
	}

	message := bs.signEd25519(e, buffer)
	return message, nil
}

func (bs *blockReductionSigner) signBLS(e *events.Reduction) error {
	signedHash, err := bls.Sign(bs.BLSSecretKey, bs.BLSPubKey, e.VotedHash)
	e.SignedHash = signedHash.Compress()
	e.Header.PubKeyBLS = bs.BLSPubKey.Marshal()
	return err
}

func (bs *blockReductionSigner) signEd25519(e *events.Reduction, eventBuf *bytes.Buffer) *bytes.Buffer {
	signature := ed25519.Sign(*bs.EdSecretKey, eventBuf.Bytes())
	buf := new(bytes.Buffer)
	if err := encoding.Write512(buf, signature); err != nil {
		panic(err)
	}

	if err := encoding.Write256(buf, bs.EdPubKeyBytes()); err != nil {
		panic(err)
	}

	if _, err := buf.Write(eventBuf.Bytes()); err != nil {
		panic(err)
	}

	return buf
}
