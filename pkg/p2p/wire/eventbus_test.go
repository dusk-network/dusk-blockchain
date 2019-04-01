package wire

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {

	eb := New()
	assert.NotNil(t, eb)
}

func TestSubscribe(t *testing.T) {
	eb := New()
	myChan := make(chan *bytes.Buffer, 10)
	assert.NotNil(t, eb.Subscribe("whateverTopic", myChan))
}

func TestPublish(t *testing.T) {
	newEB(t)
}

func TestUnsubscribe(t *testing.T) {
	eb, myChan, id := newEB(t)
	eb.Unsubscribe("whateverTopic", id)

	eb.Publish("whateverTopic", bytes.NewBufferString("whatever2"))
	select {
	case <-myChan:
		assert.FailNow(t, "We should have not received message")
	case <-time.After(50 * time.Millisecond):
		// success
	}
}

func TestUnsubscribeAll(t *testing.T) {
	eb, myChan, id := newEB(t)
	eb.UnsubscribeAll(id)
	eb.Publish("whateverTopic", bytes.NewBufferString("whatever2"))
	select {
	case <-myChan:
		assert.FailNow(t, "We should have not received message")
	case <-time.After(20 * time.Millisecond):
		// success
	}
}

func newEB(t *testing.T) (*EventBus, chan *bytes.Buffer, uint32) {
	eb := New()
	myChan := make(chan *bytes.Buffer, 10)
	id := eb.Subscribe("whateverTopic", myChan)
	assert.NotNil(t, id)

	eb.Publish("whateverTopic", bytes.NewBufferString("whatever"))

	select {
	case received := <-myChan:
		assert.Equal(t, "whatever", received.String())
	case <-time.After(50 * time.Millisecond):
		assert.FailNow(t, "We should have received a message by now")
	}

	return eb, myChan, id
}
