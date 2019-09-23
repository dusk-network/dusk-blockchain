package committee

import (
	"bytes"

	"github.com/dusk-network/dusk-blockchain/pkg/core/consensus/msg"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/eventbus"

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
	p := &provisioner{make([]byte, 32), make([]byte, 129), 0, 0, 0}
	if err := encoding.Read256(r, p.pubKeyEd); err != nil {
		return nil, err
	}

	if err := encoding.ReadVarBytes(r, &p.pubKeyBLS); err != nil {
		return nil, err
	}

	if err := encoding.ReadUint64LE(r, &p.amount); err != nil {
		return nil, err
	}

	if err := encoding.ReadUint64LE(r, &p.startHeight); err != nil {
		return nil, err
	}

	if err := encoding.ReadUint64LE(r, &p.endHeight); err != nil {
		return nil, err
	}

	return p, nil
}

func initRemoveProvisionerCollector(subscriber eventbus.Subscriber) chan []byte {
	removeProvisionerChan := make(chan []byte, 50)
	collector := &removeProvisionerCollector{removeProvisionerChan}
	go eventbus.NewTopicListener(subscriber, collector, msg.RemoveProvisionerTopic).Accept()
	return removeProvisionerChan
}

func (r *removeProvisionerCollector) Collect(key *bytes.Buffer) error {
	r.removeProvisionerChan <- key.Bytes()
	return nil
}
