package reputation

import (
	"bytes"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type absenteeCollector struct {
	absenteesChan chan [][]byte
}

func initAbsenteeCollector(eventSubscriber wire.EventSubscriber) chan [][]byte {
	absenteesChan := make(chan [][]byte, 1)
	collector := &absenteeCollector{absenteesChan}
	go wire.NewTopicListener(eventSubscriber, collector, msg.AbsenteesTopic).Accept()
	return absenteesChan
}

func (a *absenteeCollector) Collect(m *bytes.Buffer) error {
	lenAbsentees, err := encoding.ReadVarInt(m)
	if err != nil {
		return err
	}

	absentees := make([][]byte, lenAbsentees)
	for i := uint64(0); i < lenAbsentees; i++ {
		if err := encoding.ReadVarBytes(m, &absentees[i]); err != nil {
			return err
		}
	}

	a.absenteesChan <- absentees
	return nil
}
