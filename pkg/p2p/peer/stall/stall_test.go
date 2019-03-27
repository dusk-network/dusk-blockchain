package stall

import (
	"io/ioutil"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/topics"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func TestAddRemoveMessage(t *testing.T) {

	responseTime := 2 * time.Second
	tickerInterval := 1 * time.Second

	d := NewDetector(responseTime, tickerInterval)
	d.AddMessage(topics.GetAddr)
	mp := d.GetMessages()

	assert.Equal(t, 1, len(mp))
	assert.IsType(t, time.Time{}, mp[topics.GetAddr])

	d.RemoveMessage(topics.GetAddr)
	mp = d.GetMessages()

	assert.Equal(t, 0, len(mp))
	assert.Empty(t, mp[topics.GetAddr])
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
	mp.detector.lock.Lock()
	mp.online = false
	mp.detector.lock.Unlock()
}

func TestDeadlineWorks(t *testing.T) {

	responseTime := 2 * time.Second
	tickerInterval := 1 * time.Second

	d := NewDetector(responseTime, tickerInterval)
	mp := mockPeer{online: true, detector: d}

	go mp.loop()

	d.AddMessage(topics.GetAddr)
	time.Sleep(responseTime + 1*time.Second)

	k := make(map[topics.Topic]time.Time)
	d.lock.Lock()
	assert.Equal(t, k, d.responses)
	d.lock.Unlock()
	assert.Equal(t, false, mp.online)
}
func TestDeadlineShouldNotBeEmpty(t *testing.T) {
	responseTime := 10 * time.Second
	tickerInterval := 1 * time.Second

	d := NewDetector(responseTime, tickerInterval)
	d.AddMessage(topics.GetAddr)
	time.Sleep(1 * time.Second)

	k := make(map[topics.Topic]time.Time)
	assert.NotEqual(t, k, d.responses)
}
