package event

import (
	"github.com/spf13/viper"
	"sync"
	"time"
)

// ConditionalEvent is an interface that can be use
type ConditionalEvent interface {
	GetType() Type
	IsExecutable() bool
	Execute() ConditionalEvent
	setExecutableFlag()
}

// syncEvent defines a executable monitor event
type syncEvent struct {
	Type           Type
	executableFlag bool
	timeDisabled   time.Time
}

// Type defines the monitoring event types
type Type uint8

// Monitoring events
const (
	MinPeersExcd        Type = 0x01
	MaxPeersExcd        Type = 0x02
	OutOfPeerAddr       Type = 0x03
	MinGetAddrPeersExcd Type = 0x04
)

var events map[Type]ConditionalEvent
var once sync.Once

// GetEvent gets the syncEvent instance of a specific type as a Singleton
func GetEvent(evtType Type) ConditionalEvent {
	if events == nil {
		once.Do(func() {
			events = make(map[Type]ConditionalEvent, 4)
			events[MinPeersExcd] = &syncEvent{Type: MinPeersExcd, timeDisabled: time.Time{}}
			events[MaxPeersExcd] = &syncEvent{Type: MaxPeersExcd, timeDisabled: time.Time{}}
			events[OutOfPeerAddr] = &syncEvent{Type: OutOfPeerAddr, timeDisabled: time.Time{}}
			events[MinGetAddrPeersExcd] = &syncEvent{Type: MinGetAddrPeersExcd, timeDisabled: time.Time{}}
		})
	}
	evt := events[evtType]
	evt.setExecutableFlag()
	return evt
}

func (m *syncEvent) GetType() Type {
	return m.Type
}

// IsExecutable returns if the event should go off e.g. after a waiting period
func (m *syncEvent) IsExecutable() bool {
	return m.executableFlag == true
}

// Execute sets the current time and returns the event
// E.g. to put it on a channel.
func (m *syncEvent) Execute() ConditionalEvent {
	m.timeDisabled = time.Now()
	return m
}

func (m *syncEvent) setExecutableFlag() {
	elapsedTime := time.Now().Sub(m.timeDisabled)
	duration, _ := time.ParseDuration(viper.GetString("net.monitoring.evtDisableDuration"))

	if elapsedTime > duration {
		m.executableFlag = true
	} else {
		m.executableFlag = false
	}
}
