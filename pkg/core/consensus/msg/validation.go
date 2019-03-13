package msg

import (
	"bytes"
	"encoding/hex"
	"errors"

	"github.com/cretz/bine/torutil/ed25519"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/bls"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/util/nativeutils/prerror"
)

func Validate(messageBytes *bytes.Buffer) error {
	signedMessage, err := decodeSignedMessage(messageBytes)
	if err != nil {
		return err
	}

	pubKey, err := decodePubKey(messageBytes)
	if err != nil {
		return err
	}

	if !ed25519.Verify(pubKey, messageBytes.Bytes(), signedMessage) {
		return errors.New("ed25519 verification failed")
	}

	return nil
}

func decodeSignedMessage(messageBytes *bytes.Buffer) ([]byte, error) {
	var signedMessage []byte
	if err := encoding.Read512(messageBytes, &signedMessage); err != nil {
		return nil, err
	}

	return signedMessage, nil
}

func decodePubKey(messageBytes *bytes.Buffer) ([]byte, error) {
	var pubKey []byte
	if err := encoding.Read256(messageBytes, &pubKey); err != nil {
		return nil, err
	}

	return pubKey, nil
}

func VerifyVote(vote *Vote, hash []byte, step uint8,
	votingCommittee1, votingCommittee2 map[string]uint8) *prerror.PrError {
	if err := checkVoterEligibility(vote.PubKeyBLS, votingCommittee1,
		votingCommittee2); err != nil {

		return err
	}

	// A set should purely contain votes for one single hash
	if !bytes.Equal(hash, vote.VotedHash) {
		return prerror.New(prerror.Low, errors.New("voteset contains a vote "+
			"for the wrong hash"))
	}

	// A vote should be from the same step or the step before, in comparison
	// to the passed step parameter
	if notFromValidStep(vote.Step, step) {
		return prerror.New(prerror.Low, errors.New("vote is from another cycle"))
	}

	// Verify signature
	if err := VerifyBLSSignature(vote.PubKeyBLS, vote.VotedHash, vote.SignedHash); err != nil {
		return prerror.New(prerror.Low, errors.New("BLS verification failed"))
	}

	return nil
}

func checkVoterEligibility(pubKeyBLS []byte, votingCommittee1,
	votingCommittee2 map[string]uint8) *prerror.PrError {

	pubKeyStr := hex.EncodeToString(pubKeyBLS)
	if votingCommittee1[pubKeyStr] == 0 && votingCommittee2[pubKeyStr] == 0 {
		return prerror.New(prerror.Low, errors.New("voter is not eligible to vote"))
	}

	return nil
}

func notFromValidStep(voteStep, setStep uint8) bool {
	return voteStep != setStep && voteStep+1 != setStep
}

func VerifyBLSSignature(pubKeyBytes, message, signature []byte) error {
	pubKeyBLS := &bls.PublicKey{}
	if err := pubKeyBLS.Unmarshal(pubKeyBytes); err != nil {
		return err
	}

	sig := &bls.Signature{}
	if err := sig.Decompress(signature); err != nil {
		return err
	}

	apk := bls.NewApk(pubKeyBLS)
	return bls.Verify(apk, message, sig)
}
