package user

import (
	"bytes"
	"encoding/binary"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

const addProvisionerTopic = "addprovisioner"

type CommitteeStore struct {
	eventBus              *wire.EventBus
	addProvisionerChannel <-chan *bytes.Buffer
	addProvisionerID      uint32

	provisioners *Provisioners
	totalWeight  uint64
}

func NewCommitteeStore(eventBus *wire.EventBus) *CommitteeStore {
	addProvisionerChannel := make(chan *bytes.Buffer, 100)

	committeeStore := &CommitteeStore{
		eventBus:              eventBus,
		addProvisionerChannel: addProvisionerChannel,
		provisioners:          &Provisioners{},
	}

	addProvisionerID := committeeStore.eventBus.Subscribe(addProvisionerTopic,
		addProvisionerChannel)
	committeeStore.addProvisionerID = addProvisionerID

	return committeeStore
}

func (c *CommitteeStore) Listen() {
	for {
		select {
		case newProvisionerBytes := <-c.addProvisionerChannel:
			pubKeyEd, pubKeyBLS, amount, err := decodeNewProvisioner(newProvisionerBytes)
			if err != nil {
				// Log
				return
			}

			if err := c.provisioners.AddMember(pubKeyEd, pubKeyBLS, amount); err != nil {
				break
			}

			c.totalWeight += amount
			c.eventBus.Publish(msg.ProvisionerAddedTopic, newProvisionerBytes)
		}
	}
}

// TotalWeight will return the totalWeight field of the CommitteeStore.
func (c *CommitteeStore) TotalWeight() uint64 {
	return c.totalWeight
}

// Get the provisioner committee and return it
func (c *CommitteeStore) Get() Provisioners {
	return *c.provisioners
}

func decodeNewProvisioner(r io.Reader) ([]byte, []byte, uint64, error) {
	var pubKeyEd []byte
	if err := encoding.Read256(r, &pubKeyEd); err != nil {
		return nil, nil, 0, err
	}

	var pubKeyBLS []byte
	if err := encoding.ReadVarBytes(r, &pubKeyBLS); err != nil {
		return nil, nil, 0, err
	}

	var amount uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &amount); err != nil {
		return nil, nil, 0, err
	}

	return pubKeyEd, pubKeyBLS, amount, nil
}
