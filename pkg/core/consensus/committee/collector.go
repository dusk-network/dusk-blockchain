package committee

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
)

type (
	provisioner struct {
		pubKeyEd    []byte
		pubKeyBLS   []byte
		amount      uint64
		startHeight uint64
		endHeight   uint64
	}

	removeProvisionerCollector struct {
		removeProvisionerChan chan []byte
	}
)

func decodeNewProvisioner(r *bytes.Buffer) (*provisioner, error) {
	pubKeyEd, err := encoding.Read256(r)
	if err != nil {
		return nil, err
	}

	pubKeyBLS, err := encoding.ReadVarBytes(r)
	if err != nil {
		return nil, err
	}

	amount, err := encoding.ReadUint64LE(r)
	if err != nil {
		return nil, err
	}

	startHeight, err := encoding.ReadUint64LE(r)
	if err != nil {
		return nil, err
	}

	endHeight, err := encoding.ReadUint64LE(r)
	if err != nil {
		return nil, err
	}

	return &provisioner{pubKeyEd, pubKeyBLS, amount, startHeight, endHeight}, nil
}

func initRemoveProvisionerCollector(subscriber wire.EventSubscriber) chan []byte {
	removeProvisionerChan := make(chan []byte, 50)
	collector := &removeProvisionerCollector{removeProvisionerChan}
	go wire.NewTopicListener(subscriber, collector, msg.RemoveProvisionerTopic).Accept()
	return removeProvisionerChan
}

func (r *removeProvisionerCollector) Collect(key *bytes.Buffer) error {
	r.removeProvisionerChan <- key.Bytes()
	return nil
}
