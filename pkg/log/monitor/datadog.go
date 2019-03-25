package monitor

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/log"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

type DataDog struct {
	logChan chan *log.Event
}

func LaunchDog(eventBus *wire.EventBus) *DataDog {
	logChan := log.InitLogCollector(eventBus)
	dd := &DataDog{
		logChan,
	}
	go dd.Listen()
	return dd
}

func (d *DataDog) Listen() {
	for {
		ev := <-d.logChan
		d.SendToDog(ev)
	}
}

func (d *DataDog) SendToDog(ev *log.Event) {
	//do your thing
}
