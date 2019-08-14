package consensus

import (
	"bytes"
	"errors"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/initiation"
	"github.com/dusk-network/dusk-blockchain/pkg/core/transactions"
)

// InCommittee will query the blockchain for any non-expired stakes that belong to the supplied public key.
func InCommittee(blsPubKey []byte) bool {
	initiator := initiation.NewInitiator(nil, blsPubKey, FindStake)
	_, err := initiator.SearchForValue()
	return err != nil
}

func FindStake(txs []transactions.Transaction, item []byte) ([]byte, error) {
	for _, tx := range txs {
		stake, ok := tx.(*transactions.Stake)
		if !ok {
			continue
		}

		if bytes.Equal(item, stake.PubKeyBLS) {
			// When searching for stakes, we dont really need any value, we just need to know that the stake exists.
			// So, we don't return anything here.
			return nil, nil
		}
	}

	return nil, errors.New("could not find corresponding stake")
}
