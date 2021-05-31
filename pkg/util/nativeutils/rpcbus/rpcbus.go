// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this
// file, you can obtain one at https://opensource.org/licenses/MIT.
//
// Copyright (c) DUSK NETWORK. All rights reserved.

package rpcbus

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

// ErrMethodNotExists is returned when calling an unregistered method.
type ErrMethodNotExists struct {
	topic topics.Topic
}

// Error obeys the error interface.
func (e *ErrMethodNotExists) Error() string {
	return fmt.Sprintf("method not registered: %s", e.topic)
}

var (
	// ErrRequestTimeout is returned when request timeout-ed.
	ErrRequestTimeout = errors.New("timeout-ed request")

	// ErrMethodExists is returned when method is already registered.
	ErrMethodExists = errors.New("method exists already")

	// ErrInvalidRequestChan is returned method is bound to nil chan.
	ErrInvalidRequestChan = errors.New("invalid request channel")

	// DefaultTimeout is used when 0 timeout is set on calling a method.
	DefaultTimeout = 5 * time.Second
)

// RPCBus is a request–response mechanism for internal communication between node
// components/subsystems. Under the hood this is long-polling method based on
// "chan chan" technique.
type RPCBus struct {
	mu       sync.RWMutex
	registry map[topics.Topic]chan<- Request
}

// Request is the request object forwarded to the RPC endpoint.
type Request struct {
	Params   interface{}
	RespChan chan Response
}

// Response is the response to the request forwarded to the RPC endpoint.
type Response struct {
	Resp interface{}
	Err  error
}

// EmptyRequest returns a Request instance with no parameters.
func EmptyRequest() Request {
	return NewRequest(bytes.Buffer{})
}

// NewRequest builds a new request with params. It creates the response channel
// under the hood.
func NewRequest(p interface{}) Request {
	return Request{
		Params:   p,
		RespChan: make(chan Response, 1),
	}
}

// NewResponse builds a new response.
func NewResponse(p interface{}, err error) Response {
	return Response{
		Resp: p,
		Err:  err,
	}
}

// New creates an RPCBus instance.
func New() *RPCBus {
	return &RPCBus{
		registry: make(map[topics.Topic]chan<- Request),
	}
}

// Register registers a method and binds it to a handler channel. methodName
// must be unique per node instance. If not, returns err.
func (bus *RPCBus) Register(t topics.Topic, req chan<- Request) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	if req == nil {
		return ErrInvalidRequestChan
	}

	if _, ok := bus.registry[t]; ok {
		return ErrMethodExists
	}

	bus.registry[t] = req
	return nil
}

// Deregister removes a handler channel from a method.
func (bus *RPCBus) Deregister(t topics.Topic) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	delete(bus.registry, t)
}

// Call runs a long-polling technique to request from the method Consumer to
// run the corresponding procedure and return a result or timeout.
func (bus *RPCBus) Call(t topics.Topic, req Request, timeOut time.Duration) (interface{}, error) {
	reqChan, err := bus.getReqChan(t)
	if err != nil {
		return bytes.Buffer{}, err
	}

	if timeOut <= 0 {
		timeOut = DefaultTimeout
	}

	return bus.callTimeout(reqChan, req, timeOut)
}

func (bus *RPCBus) callTimeout(reqChan chan<- Request, req Request, timeOut time.Duration) (interface{}, error) {
	timer := time.NewTimer(timeOut)

	select {
	case reqChan <- req:
	case <-timer.C:
		return bytes.Buffer{}, ErrRequestTimeout
	}

	if !timer.Stop() {
		<-timer.C
	}

	timer.Reset(timeOut)

	var resp Response
	select {
	case resp = <-req.RespChan:
	case <-timer.C:
		return bytes.Buffer{}, ErrRequestTimeout
	}

	if !timer.Stop() {
		<-timer.C
	}

	return resp.Resp, resp.Err
}

//nolint
func (bus *RPCBus) callNoTimeout(reqChan chan<- Request, req Request) (interface{}, error) {
	reqChan <- req
	resp := <-req.RespChan
	return resp.Resp, resp.Err
}

func (bus *RPCBus) getReqChan(t topics.Topic) (chan<- Request, error) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	if reqChan, ok := bus.registry[t]; ok {
		return reqChan, nil
	}

	return nil, &ErrMethodNotExists{t}
}

// Close the RPCBus by resetting the registry.
func (bus *RPCBus) Close() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	// Channels should be closed only by the goroutines/components
	// that make them

	// Reset registry
	bus.registry = make(map[topics.Topic]chan<- Request)
}
