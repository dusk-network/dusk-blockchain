package monitor

import (
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/log"
)

type DataDog struct {
}

func (d *DataDog) Connect(logChan <-chan *log.Event) {
	go func() {
		for {
			ev := <-logChan
			d.SendToDog(ev)
		}
	}()
}

func (d *DataDog) SendToDog(ev *log.Event) {
	//do your thing
}
