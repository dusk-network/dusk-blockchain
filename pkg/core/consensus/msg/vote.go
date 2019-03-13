package msg

import (
	"bytes"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/hash"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Vote defines a block set or signature set vote for the consensus.
type Vote struct {
	VotedHash  []byte // The block hash or signature set hash voted on
	PubKeyBLS  []byte // BLS public key of the voter
	SignedHash []byte // Compressed BLS signature of the hash
	Step       uint8  // Step this vote occurred at
}

func DecodeVoteSet(r io.Reader) ([]*Vote, error) {
	lVotes, err := encoding.ReadVarInt(r)
	if err != nil {
		return nil, err
	}

	voteSet := make([]*Vote, lVotes)
	for i := uint64(0); i < lVotes; i++ {
		voteSet[i], err = decodeVote(r)
		if err != nil {
			return nil, err
		}
	}

	return voteSet, nil
}

// DecodeVote will a Vote struct from r and return it.
func decodeVote(r io.Reader) (*Vote, error) {
	var votedHash []byte
	if err := encoding.Read256(r, &votedHash); err != nil {
		return nil, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
		return nil, err
	}

	var signedHash []byte
	if err := encoding.ReadBLS(r, &signedHash); err != nil {
		return nil, err
	}

	var step uint8
	if err := encoding.ReadUint8(r, &step); err != nil {
		return nil, err
	}

	return &Vote{
		VotedHash:  votedHash,
		PubKeyBLS:  pubKeyBLS,
		SignedHash: signedHash,
		Step:       step,
	}, nil
}

func HashVoteSet(voteSet []*Vote) ([]byte, error) {
	encodedVoteSet, err := EncodeVoteSet(voteSet)
	if err != nil {
		return nil, err
	}

	// Hash bytes
	sigSetHash, err := hash.Sha3256(encodedVoteSet)
	if err != nil {
		return nil, err
	}

	return sigSetHash, nil
}

func EncodeVoteSet(voteSet []*Vote) ([]byte, error) {
	buffer := new(bytes.Buffer)
	if err := encoding.WriteVarInt(buffer, uint64(len(voteSet))); err != nil {
		return nil, err
	}

	for _, vote := range voteSet {
		if err := encodeVote(vote, buffer); err != nil {
			return nil, err
		}
	}

	return buffer.Bytes(), nil
}

func encodeVote(vote *Vote, w io.Writer) error {
	if err := encoding.Write256(w, vote.VotedHash); err != nil {
		return err
	}

	if err := encoding.WriteVarBytes(w, vote.PubKeyBLS); err != nil {
		return err
	}

	if err := encoding.WriteBLS(w, vote.SignedHash); err != nil {
		return err
	}

	if err := encoding.WriteUint8(w, vote.Step); err != nil {
		return err
	}

	return nil
}
