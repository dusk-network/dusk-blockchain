package transactions

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/data/ipc/common"
	"github.com/dusk-network/dusk-protobuf/autogen/go/rusk"
)

// Crossover is the crossover note used in a Phoenix transaction.
type Crossover struct {
	ValueComm     *common.JubJubCompressed `json:"value_comm"`
	Nonce         *common.BlsScalar        `json:"nonce"`
	EncryptedData *common.PoseidonCipher   `json:"encrypted_data"`
}

// NewCrossover returns a new empty Crossover struct.
func NewCrossover() *Crossover {
	return &Crossover{
		ValueComm:     common.NewJubJubCompressed(),
		Nonce:         common.NewBlsScalar(),
		EncryptedData: common.NewPoseidonCipher(),
	}
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
	r.ValueComm = new(rusk.JubJubCompressed)
	common.MJubJubCompressed(r.ValueComm, f.ValueComm)
	r.Nonce = new(rusk.BlsScalar)
	common.MBlsScalar(r.Nonce, f.Nonce)
	// XXX: fix typo in rusk-schema
	r.EncyptedData = new(rusk.PoseidonCipher)
	common.MPoseidonCipher(r.EncyptedData, f.EncryptedData)
}

// UCrossover copies the Rusk Crossover structure into the native equivalent.
func UCrossover(r *rusk.Crossover, f *Crossover) {
	f.ValueComm = new(common.JubJubCompressed)
	common.UJubJubCompressed(r.ValueComm, f.ValueComm)
	f.Nonce = new(common.BlsScalar)
	common.UBlsScalar(r.Nonce, f.Nonce)
	// XXX: fix typo in rusk-schema
	f.EncryptedData = new(common.PoseidonCipher)
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
	if err := common.UnmarshalJubJubCompressed(r, f.ValueComm); err != nil {
		return err
	}

	if err := common.UnmarshalBlsScalar(r, f.Nonce); err != nil {
		return err
	}

	return common.UnmarshalPoseidonCipher(r, f.EncryptedData)
}

// Equal returns whether or not two Crossovers are equal.
func (c *Crossover) Equal(other *Crossover) bool {
	if !c.ValueComm.Equal(other.ValueComm) {
		return false
	}

	if !c.Nonce.Equal(other.Nonce) {
		return false
	}

	return c.EncryptedData.Equal(other.EncryptedData)
}
