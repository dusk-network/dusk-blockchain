package rpcbus

import (
	"bytes"
	"errors"
	"sync"
	"time"
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
	registry map[method]chan<- Request
}

type Request struct {
	Params   bytes.Buffer
	Timeout  int
	RespChan chan Response
}

type Response struct {
	Resp bytes.Buffer
	Err  error
}

func New() *RPCBus {
	return &RPCBus{
		registry: make(map[method]chan<- Request),
	}
}

// Register registers a method and binds it to a handler channel. methodName
// must be unique per node instance. if not, returns err
func (bus *RPCBus) Register(m method, req chan<- Request) error {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	if req == nil {
		return ErrInvalidRequestChan
	}

	if _, ok := bus.registry[m]; ok {
		return ErrMethodExists
	}

	bus.registry[m] = req
	return nil
}

// Call runs a long-polling technique to request from the method Consumer to
// run the corresponding procedure and return a result or timeout
func (bus *RPCBus) Call(m method, req Request) (bytes.Buffer, error) {
	if req.Timeout > 0 {
		return bus.callTimeout(m, req)
	}

	return bus.callNoTimeout(m, req)
}

func (bus *RPCBus) callTimeout(m method, req Request) (bytes.Buffer, error) {
	reqChan, err := bus.getReqChan(m)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Send the request with write-timeout
	select {
	case reqChan <- req:
	case <-time.After(time.Duration(req.Timeout) * time.Second):
		return bytes.Buffer{}, ErrRequestTimeout
	}

	// Wait for response or err from the consumer with read-timeout
	var resp Response
	select {
	case resp = <-req.RespChan:
	case <-time.After(time.Duration(req.Timeout) * time.Second):
		err = ErrRequestTimeout
	}

	return resp.Resp, resp.Err
}

func (bus *RPCBus) callNoTimeout(m method, req Request) (bytes.Buffer, error) {
	reqChan, err := bus.getReqChan(m)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Send the request with write-timeout
	reqChan <- req

	// Wait for response or err from the consumer with read-timeout
	resp := <-req.RespChan

	return resp.Resp, resp.Err
}

// NewRequest builds a new request with params
// if timeout is not positive, the call waits infinitely
func NewRequest(p bytes.Buffer, timeout int) Request {
	d := Request{Timeout: timeout, Params: p}
	d.RespChan = make(chan Response, 1)
	return d
}

func (bus *RPCBus) getReqChan(m method) (chan<- Request, error) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	if reqChan, ok := bus.registry[m]; ok {
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
