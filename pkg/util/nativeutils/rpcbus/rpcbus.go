package rpcbus

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

var (
	// ErrRequestTimeout is returned when request timeout-ed
	ErrRequestTimeout = errors.New("timeout-ed request")

	// ErrMethodExists is returned when method is already registered
	ErrMethodExists = errors.New("method exists already")

	// ErrMethodNotExists is returned when calling an unregistered method
	ErrMethodNotExists = errors.New("method not registered")

	// ErrInvalidRequestChan is returned method is bound to nil chan
	ErrInvalidRequestChan = errors.New("invalid request channel")
)

// RPCBus is a request–response mechanism for internal communication between node
// components/subsystems. Under the hood this is long-polling method based on
// "chan chan" technique.
type RPCBus struct {
	mu       sync.RWMutex
	registry map[topics.Topic]chan<- Request
}

type Request struct {
	Params   interface{}
	RespChan chan Response
}

type Response struct {
	Resp interface{}
	Err  error
}

func EmptyRequest() Request {
	return NewRequest(bytes.Buffer{})
}

// NewRequest builds a new request with params
func NewRequest(p interface{}) Request {
	return Request{
		Params:   p,
		RespChan: make(chan Response, 1),
	}
}

func New() *RPCBus {
	return &RPCBus{
		registry: make(map[topics.Topic]chan<- Request),
	}
}

// Register registers a method and binds it to a handler channel. methodName
// must be unique per node instance. if not, returns err
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

// Call runs a long-polling technique to request from the method Consumer to
// run the corresponding procedure and return a result or timeout
func (bus *RPCBus) Call(t topics.Topic, req Request, timeOut time.Duration) (interface{}, error) {
	reqChan, err := bus.getReqChan(t)
	if err != nil {
		return bytes.Buffer{}, err
	}

	if timeOut > 0 {
		return bus.callTimeout(reqChan, req, timeOut)
	}

	return bus.callNoTimeout(reqChan, req)
}

func (bus *RPCBus) callTimeout(reqChan chan<- Request, req Request, timeOut time.Duration) (interface{}, error) {
	select {
	case reqChan <- req:
	case <-time.After(timeOut):
		return bytes.Buffer{}, ErrRequestTimeout
	}

	var resp Response
	select {
	case resp = <-req.RespChan:
	case <-time.After(timeOut):
		return bytes.Buffer{}, ErrRequestTimeout
	}

	return resp.Resp, resp.Err
}

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

	return nil, ErrMethodNotExists
}

// Close all open channels
func (bus *RPCBus) Close() {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	for _, m := range bus.registry {
		close(m)
	}

	bus.registry = nil
}
