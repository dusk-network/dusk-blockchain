package agreement

import (
	"bytes"
	"errors"
	"fmt"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/committee"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/header"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/reduction"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/user"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/sortedset"
	"golang.org/x/crypto/ed25519"
)

type agreementHandler struct {
	user.Keys
	committee.Foldable
	*UnMarshaller
}

// newHandler returns an initialized agreementHandler.
func newHandler(committee committee.Foldable, keys user.Keys) *agreementHandler {
	return &agreementHandler{
		Keys:         keys,
		Foldable:     committee,
		UnMarshaller: NewUnMarshaller(),
	}
}

// AmMember checks if we are part of the committee.
func (a *agreementHandler) AmMember(round uint64, step uint8) bool {
	return a.Foldable.IsMember(a.Keys.BLSPubKeyBytes, round, step)
}

func (a *agreementHandler) ExtractHeader(e wire.Event) *header.Header {
	ev := e.(*Agreement)
	return &header.Header{
		Round: ev.Round,
		Step:  ev.Step,
	}
}

func (a *agreementHandler) ExtractIdentifier(e wire.Event, r *bytes.Buffer) error {
	ev := e.(*Agreement)
	return encoding.WriteUint8(r, ev.Step)
}

// Verify checks the signature of the set.
func (a *agreementHandler) Verify(e wire.Event) error {
	ev, ok := e.(*Agreement)
	if !ok {
		return errors.New("Cant' verify an event different than the aggregated agreement")
	}
	allVoters := 0
	for i, votes := range ev.VotesPerStep {
		step := uint8(int(ev.Step*2) + (i - 1)) // the event step is the second one of the reduction cycle
		subcommittee := a.Unpack(votes.BitSet, ev.Round, step)
		allVoters += len(subcommittee)
		apk, err := ReconstructApk(subcommittee)
		if err != nil {
			return err
		}

		if err := VerifySignatures(ev.Round, step, ev.BlockHash, apk, votes.Signature); err != nil {
			return err
		}
	}

	if allVoters < a.Quorum(ev.Round) {
		return fmt.Errorf("vote set too small - %v/%v", allVoters, a.Quorum(ev.Round))
	}
	return nil
}

// ReconstructApk reconstructs an aggregated BLS public key from a subcommittee.
func ReconstructApk(subcommittee sortedset.Set) (*bls.Apk, error) {
	var apk *bls.Apk
	if len(subcommittee) == 0 {
		return nil, errors.New("Subcommittee is empty")
	}
	for i, ipk := range subcommittee {
		pk, err := bls.UnmarshalPk(ipk.Bytes())
		if err != nil {
			return nil, err
		}
		if i == 0 {
			apk = bls.NewApk(pk)
			continue
		}
		if err := apk.Aggregate(pk); err != nil {
			return nil, err
		}
	}

	return apk, nil
}

func VerifySignatures(round uint64, step uint8, blockHash []byte, apk *bls.Apk, sig *bls.Signature) error {
	signed := new(bytes.Buffer)
	vote := &header.Header{
		Round:     round,
		Step:      step,
		BlockHash: blockHash,
	}

	if err := header.MarshalSignableVote(signed, vote); err != nil {
		return err
	}

	return bls.Verify(apk, signed.Bytes(), sig)
}

func (a *agreementHandler) signEd25519(eventBuf *bytes.Buffer) *bytes.Buffer {
	signature := ed25519.Sign(*a.EdSecretKey, eventBuf.Bytes())
	buf := new(bytes.Buffer)
	if err := encoding.Write512(buf, signature); err != nil {
		panic(err)
	}

	if err := encoding.Write256(buf, a.EdPubKeyBytes); err != nil {
		panic(err)
	}

	if _, err := buf.Write(eventBuf.Bytes()); err != nil {
		panic(err)
	}

	return buf
}

func (a *agreementHandler) createAgreement(evs []wire.Event, round uint64, step uint8) (*bytes.Buffer, error) {
	rev := evs[0].(*reduction.Reduction)
	h := &header.Header{
		PubKeyBLS: a.BLSPubKeyBytes,
		Round:     round,
		Step:      step,
		BlockHash: rev.BlockHash,
	}

	// create the Agreement event
	aev, err := a.Aggregate(h, evs)
	if err != nil {
		return nil, err
	}

	// BLS sign it
	if err := Sign(aev, a.Keys); err != nil {
		return nil, err
	}

	// Marshall it for Ed25519 signature
	buffer := new(bytes.Buffer)
	if err := a.Marshal(buffer, aev); err != nil {
		return nil, err
	}

	// sign the whole message
	signed := a.signEd25519(buffer)

	// add the topic
	msg, err := wire.AddTopic(signed, topics.Agreement)
	if err != nil {
		return nil, err
	}

	//send it
	return msg, nil
}

// Aggregate the Agreement event into an Agreement outgoing event
func (a *agreementHandler) Aggregate(h *header.Header, voteSet []wire.Event) (*Agreement, error) {
	stepVotesMap := make(map[uint8]struct {
		*StepVotes
		sortedset.Set
	})

	for _, ev := range voteSet {
		reduction := ev.(*reduction.Reduction)
		sv, found := stepVotesMap[reduction.Step]
		if !found {
			sv.StepVotes = NewStepVotes()
			sv.Set = sortedset.New()
		}

		if err := sv.StepVotes.Add(reduction.SignedHash, reduction.Sender(), reduction.Step); err != nil {
			return nil, err
		}
		sv.Set.Insert(reduction.PubKeyBLS)
		stepVotesMap[reduction.Step] = sv
	}

	aev := New()
	aev.Header = h
	for step, stepVotes := range stepVotesMap {
		sv, provisioners := stepVotes.StepVotes, stepVotes.Set
		sv.BitSet = a.Pack(provisioners, h.Round, sv.Step)
		if step%2 == 0 {
			aev.VotesPerStep[1] = sv
		} else {
			aev.VotesPerStep[0] = sv
		}
	}

	return aev, nil
}
