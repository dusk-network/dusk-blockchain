package committee

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
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
	var pubKeyEd []byte
	if err := encoding.Read256(r, &pubKeyEd); err != nil {
		return nil, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
		return nil, err
	}

	var amount uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &amount); err != nil {
		return nil, err
	}

	var startHeight uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &startHeight); err != nil {
		return nil, err
	}

	var endHeight uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &endHeight); err != nil {
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
