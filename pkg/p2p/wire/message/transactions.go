package message

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/transactions"
)

// UnmarshalTxMessage unmarshals a Message carrying a tx from a buffer
func UnmarshalTxMessage(r *bytes.Buffer, m SerializableMessage) error {
	cc, err := transactions.Unmarshal(r)
	if err != nil {
		return err
	}
	m.SetPayload(cc)
	return nil
}
