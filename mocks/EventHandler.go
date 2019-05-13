// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import bytes "bytes"

import events "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/consensus/events"
import mock "github.com/stretchr/testify/mock"
import wire "gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"

// EventHandler is an autogenerated mock type for the EventHandler type
type EventHandler struct {
	mock.Mock
}

// Deserialize provides a mock function with given fields: _a0
func (_m *EventHandler) Deserialize(_a0 *bytes.Buffer) (wire.Event, error) {
	ret := _m.Called(_a0)

	var r0 wire.Event
	if rf, ok := ret.Get(0).(func(*bytes.Buffer) wire.Event); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(wire.Event)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*bytes.Buffer) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExtractHeader provides a mock function with given fields: _a0
func (_m *EventHandler) ExtractHeader(_a0 wire.Event) *events.Header {
	ret := _m.Called(_a0)

	var r0 *events.Header
	if rf, ok := ret.Get(0).(func(wire.Event) *events.Header); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*events.Header)
		}
	}

	return r0
}

// Marshal provides a mock function with given fields: _a0, _a1
func (_m *EventHandler) Marshal(_a0 *bytes.Buffer, _a1 wire.Event) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*bytes.Buffer, wire.Event) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Verify provides a mock function with given fields: _a0
func (_m *EventHandler) Verify(_a0 wire.Event) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(wire.Event) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
