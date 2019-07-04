package generation

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/agreement"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type (
	agreementEventCollector struct {
		agreementEventChan chan<- wire.Event
	}

	hashCollector struct {
		hashChannel chan []byte
	}
)

func initAgreementEventCollector(subscriber wire.EventSubscriber) <-chan wire.Event {
	agreementEventChan := make(chan wire.Event, 1)
	collector := &agreementEventCollector{agreementEventChan}
	go wire.NewTopicListener(subscriber, collector, msg.AgreementEventTopic).Accept()
	return agreementEventChan
}

func (a *agreementEventCollector) Collect(m *bytes.Buffer) error {
	unmarshaller := agreement.NewUnMarshaller()
	ev, err := unmarshaller.Deserialize(m)
	if err != nil {
		return err
	}

	a.agreementEventChan <- ev
	return nil
}

func initWinningHashCollector(subscriber wire.EventSubscriber) chan []byte {
	hashChannel := make(chan []byte, 1)
	collector := &hashCollector{hashChannel}
	go wire.NewTopicListener(subscriber, collector, msg.WinningBlockHashTopic).Accept()
	return hashChannel
}

func (c *hashCollector) Collect(message *bytes.Buffer) error {
	c.hashChannel <- message.Bytes()
	return nil
}
