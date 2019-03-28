package monitor

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/msg"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gopkg.in/zorkian/go-datadog-api.v2"
)

func TestRoundUpdate(t *testing.T) {

	mChan := make(chan datadog.Metric)
	eventBus := wire.New()
	mc := &mockClient{mChan}
	roundChan := consensus.InitRoundUpdate(eventBus)
	go launchBlockTimeMonitor(eventBus, mc, roundChan)
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(78))
	// first time we are not supposed to send any blocktime since there is no duration
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(b))
	//second time should trigger the duration as a time difference between the two rounds
	<-time.After(1 * time.Second)
	eventBus.Publish(msg.RoundUpdateTopic, bytes.NewBuffer(b))
	metric := <-mChan
	obtained := metric.Points[0][1]

	assert.InDelta(t, 1, *obtained, 0.01)
}

type mockClient struct {
	mChan chan datadog.Metric
}

func (m *mockClient) PostMetrics(metrics []datadog.Metric) error {
	m.mChan <- metrics[0]
	return nil
}
