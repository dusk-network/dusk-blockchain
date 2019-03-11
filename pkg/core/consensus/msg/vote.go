package msg

import (
	"io"

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
