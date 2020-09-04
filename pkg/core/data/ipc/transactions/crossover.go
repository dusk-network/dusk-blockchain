package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

type Crossover struct {
	ValueComm     *common.JubJubCompressed `json:"value_comm"`
	Nonce         *common.BlsScalar        `json:"nonce"`
	EncryptedData *common.PoseidonCipher   `json:"encrypted_data"`
}

// Copy complies with message.Safe interface. It returns a deep copy of
// the message safe to publish to multiple subscribers.
func (c *Crossover) Copy() *Crossover {
	return &Crossover{
		ValueComm:     c.ValueComm.Copy(),
		Nonce:         c.Nonce.Copy(),
		EncryptedData: c.EncryptedData.Copy(),
	}
}

// MCrossover copies the Crossover structure into the Rusk equivalent.
func MCrossover(r *rusk.Crossover, f *Crossover) {
	common.MJubJubCompressed(r.ValueComm, f.ValueComm)
	common.MBlsScalar(r.Nonce, f.Nonce)
	// XXX: fix typo in rusk-schema
	common.MPoseidonCipher(r.EncyptedData, f.EncryptedData)
}

// UCrossover copies the Rusk Crossover structure into the native equivalent.
func UCrossover(r *rusk.Crossover, f *Crossover) {
	common.UJubJubCompressed(r.ValueComm, f.ValueComm)
	common.UBlsScalar(r.Nonce, f.Nonce)
	// XXX: fix typo in rusk-schema
	common.UPoseidonCipher(r.EncyptedData, f.EncryptedData)
}

// MarshalCrossover writes the Crossover struct into a bytes.Buffer.
func MarshalCrossover(r *bytes.Buffer, f *Crossover) error {
	if err := common.MarshalJubJubCompressed(r, f.ValueComm); err != nil {
		return err
	}

	if err := common.MarshalBlsScalar(r, f.Nonce); err != nil {
		return err
	}

	return common.MarshalPoseidonCipher(r, f.EncryptedData)
}

// UnmarshalCrossover reads a Crossover struct from a bytes.Buffer.
func UnmarshalCrossover(r *bytes.Buffer, f *Crossover) error {
	f = new(Crossover)

	if err := common.UnmarshalJubJubCompressed(r, f.ValueComm); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.Nonce); err != nil {
		return err
	}

	return common.UnmarshalPoseidonCipher(r, f.EncryptedData)
}
