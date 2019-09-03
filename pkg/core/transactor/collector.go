package transactor

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/encoding"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

type (
	transferCollector struct {
		transferChan chan transferInfo
	}

	transferInfo struct {
		amount  uint64
		address string
	}
)

func initTransferCollector(subscriber wire.EventSubscriber) chan transferInfo {
	transferChan := make(chan transferInfo, 1)
	collector := &transferCollector{transferChan}
	go wire.NewTopicListener(subscriber, collector, string(topics.Transfer)).Accept()
	return transferChan
}

func (t *transferCollector) Collect(m *bytes.Buffer) error {
	var amount uint64
	if err := encoding.ReadUint64(m, binary.LittleEndian, &amount); err != nil {
		return err
	}

	t.transferChan <- transferInfo{amount, string(m.Bytes())}
	return nil
}
