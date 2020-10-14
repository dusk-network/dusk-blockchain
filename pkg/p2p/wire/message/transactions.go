package message

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/transactions"
)

// UnmarshalTxMessage unmarshals a Message carrying a tx from a buffer
func UnmarshalTxMessage(r *bytes.Buffer, m SerializableMessage) error {
	cc := transactions.NewTransaction()
	if err := transactions.Unmarshal(r, cc); err != nil {
		return err
	}
	m.SetPayload(cc)
	return nil
}
