package reputation

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

type absentees struct {
	round uint64
	pks   [][]byte
}

func decodeAbsentees(m *bytes.Buffer) (*absentees, error) {
	var round uint64
	if err := encoding.ReadUint64LE(m, &round); err != nil {
		return nil, err
	}

	lenAbsentees, err := encoding.ReadVarInt(m)
	if err != nil {
		return nil, err
	}

	pks := make([][]byte, lenAbsentees)
	for i := range pks {
		if err = encoding.ReadVarBytes(m, &pks[i]); err != nil {
			return nil, err
		}
	}

	return &absentees{round, pks}, nil
}
