package reputation

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type absenteeCollector struct {
	absenteesChan chan []byte
}

func initAbsenteeCollector(eventSubscriber wire.EventSubscriber) chan []byte {
	absenteesChan := make(chan []byte, 50)
	collector := &absenteeCollector{absenteesChan}
	go wire.NewTopicListener(eventSubscriber, collector, msg.AbsenteesTopic).Accept()
	return absenteesChan
}

func (a *absenteeCollector) Collect(m *bytes.Buffer) error {
	a.absenteesChan <- m.Bytes()
	return nil
}
