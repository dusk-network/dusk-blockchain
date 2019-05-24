package reputation

import (
	"bytes"
	"encoding/binary"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type (
	absentees struct {
		round uint64
		pks   [][]byte
	}

	absenteeCollector struct {
		absenteesChan chan<- absentees
	}
)

func initAbsenteeCollector(eventSubscriber wire.EventSubscriber) <-chan absentees {
	absenteesChan := make(chan absentees, 1)
	collector := &absenteeCollector{absenteesChan}
	go wire.NewTopicListener(eventSubscriber, collector, msg.AbsenteesTopic).Accept()
	return absenteesChan
}

func (a *absenteeCollector) Collect(m *bytes.Buffer) error {
	var round uint64
	if err := encoding.ReadUint64(m, binary.LittleEndian, &round); err != nil {
		return err
	}

	lenAbsentees, err := encoding.ReadVarInt(m)
	if err != nil {
		return err
	}

	pks := make([][]byte, lenAbsentees)
	for i := range pks {
		if err := encoding.Read256(m, &pks[i]); err != nil {
			return err
		}
	}

	a.absenteesChan <- absentees{round, pks}
	return nil
}
