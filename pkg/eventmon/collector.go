package eventmon

import (
	"bytes"
	"time"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

type Logger interface {
	Connect(<-chan *Event)
}

// LaunchLoggers should do a plugin lookup and use the Connect method. For now, it accepts Loggers as a parameter and the plugging is delegated to the caller until the plugin system will be ready
func LaunchLoggers(eventBus *wire.EventBus, l []Logger) {
	logChan := InitLogCollector(eventBus)
	for _, logger := range l {
		logger.Connect(logChan)
	}
}

const (
	LogTopic = "LOG"

	Info uint8 = iota
	Warn uint8 = iota
	Err  uint8 = iota
)

type (
	collector struct {
		logChan chan *Event
	}

	Event struct {
		Msg        string
		Severity   uint8
		Originator string
		Time       time.Time
	}

	UnMarshaller struct{}
)

func NewEvent(sender string) *Event {
	return &Event{
		Originator: sender,
		Time:       time.Now(),
	}
}

func (e *Event) Sender() []byte {
	bs := bytes.NewBufferString(e.Originator)
	return bs.Bytes() //this should be the IP Address
}

func (e *Event) Equal(ev wire.Event) bool {
	other, ok := ev.(*Event)
	if !ok {
		return false
	}
	return other == e
}

func (eu *UnMarshaller) Marshal(b *bytes.Buffer, e wire.Event) error {
	ev := e.(*Event)

	if err := encoding.WriteUint8(b, ev.Severity); err != nil {
		return err
	}

	if err := encoding.WriteString(b, ev.Msg); err != nil {
		return err
	}

	if err := encoding.WriteString(b, ev.Originator); err != nil {
		return err
	}
	return nil
}

func (eu *UnMarshaller) Unmarshal(b *bytes.Buffer, e wire.Event) error {
	ev := e.(*Event)

	if err := encoding.ReadUint8(b, &ev.Severity); err != nil {
		return err
	}

	if err := encoding.ReadString(b, &ev.Msg); err != nil {
		return err
	}

	if err := encoding.ReadString(b, &ev.Originator); err != nil {
		return err
	}

	return nil
}

func (c *collector) Collect(b *bytes.Buffer) error {
	ev := &Event{}
	unmarshaller := &UnMarshaller{}

	_ = unmarshaller.Unmarshal(b, ev)
	c.logChan <- ev
	return nil
}

func InitLogCollector(eventBus *wire.EventBus) chan *Event {
	logChan := make(chan *Event, 100)
	collector := &collector{logChan}
	go wire.NewTopicListener(eventBus, collector, LogTopic).Accept()
	return logChan
}
