package stall

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/commands"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestAddRemoveMessage(t *testing.T) {

	responseTime := 2 * time.Second
	tickerInterval := 1 * time.Second

	d := NewDetector(responseTime, tickerInterval)
	d.AddMessage(commands.GetAddr)
	mp := d.GetMessages()

	assert.Equal(t, 1, len(mp))
	assert.IsType(t, time.Time{}, mp[commands.GetAddr])

	d.RemoveMessage(commands.GetAddr)
	mp = d.GetMessages()

	assert.Equal(t, 0, len(mp))
	assert.Empty(t, mp[commands.GetAddr])
}

type mockPeer struct {
	online   bool
	detector *Detector
}

func (mp *mockPeer) loop() {
loop:
	for {
		select {
		case <-mp.detector.Quitch:

			break loop
		}
	}
	// cleanup
	mp.online = false
}
func TestDeadlineWorks(t *testing.T) {

	responseTime := 2 * time.Second
	tickerInterval := 1 * time.Second

	d := NewDetector(responseTime, tickerInterval)
	mp := mockPeer{online: true, detector: d}
	go mp.loop()

	d.AddMessage(commands.GetAddr)
	time.Sleep(responseTime + 1*time.Second)

	k := make(map[commands.Cmd]time.Time)
	assert.Equal(t, k, d.responses)
	assert.Equal(t, false, mp.online)

}
func TestDeadlineShouldNotBeEmpty(t *testing.T) {
	responseTime := 10 * time.Second
	tickerInterval := 1 * time.Second

	d := NewDetector(responseTime, tickerInterval)
	d.AddMessage(commands.GetAddr)
	time.Sleep(1 * time.Second)

	k := make(map[commands.Cmd]time.Time)
	assert.NotEqual(t, k, d.responses)
}
