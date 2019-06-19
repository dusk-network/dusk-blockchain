package wire

import (
	"bytes"
	"errors"
	"sync"
	"time"
)

var (
	// ErrReqTimeout is returned when request timeout-ed
	ErrReqTimeout = errors.New("timeout-ed request")

	// ErrMethodExists is returned when method is already registered
	ErrMethodExists = errors.New("method exists already")

	// ErrMethodNotExists is returned when calling an unregistered method
	ErrMethodNotExists = errors.New("method not registered")

	// ErrInvalidReqChan is returned method is bound to nil chan
	ErrInvalidReqChan = errors.New("invalid request channel")
)

var (

	// Default set of registered methods

	// Provide the last/highest block from the local chain state
	// Can be implemented by Chain pkg or Database pkg.
	// Returns block.Block marshaled
	GetLastBlock     = "getLastBlock"
	GetLastBlockChan chan Req

	// Provide the list of verified txs ready to be included into next block
	//
	// Implemented by mempool
	GetVerifiedTxs     = "getVerifiedTxs"
	GetVerifiedTxsChan chan Req

	// Verify a specified candidate block
	//
	// Used by the reduction component.
	VerifyCandidateBlock     = "verifyCandidateBlock"
	VerifyCandidateBlockChan chan Req
)

// RPCBus is a request–response mechanism for internal communication between node
// components/subsystems. Under the hood this is long-polling method based on
// "chan chan" technique.
//
//
// Idiomatic communication based on `chan chan` avoiding the need of a
// mutex-per-subsystem to guard the shared state.
//
// Producer and Consumer are decoupled to avoid cross-referencing / cyclic
// dependencies issues.
//
type RPCBus struct {
	registry map[string]method
	mu       sync.RWMutex
}

type method struct {
	Name string
	req  chan<- Req
}

type Req struct {
	Params   bytes.Buffer
	Timeout  int
	RespChan chan bytes.Buffer
	ErrChan  chan error
}

func NewRPCBus() *RPCBus {
	var bus RPCBus
	bus.registry = make(map[string]method)

	// default methods

	if GetLastBlockChan == nil {
		GetLastBlockChan = make(chan Req)
		if err := bus.Register(GetLastBlock, GetLastBlockChan); err != nil {
			panic(err)
		}
	}

	if GetVerifiedTxsChan == nil {
		GetVerifiedTxsChan = make(chan Req)
		if err := bus.Register(GetVerifiedTxs, GetVerifiedTxsChan); err != nil {
			panic(err)
		}
	}

	return &bus
}

// Register registers a method and binds it to a handler channel. methodName
// must be unique per node instance. if not, returns err
func (bus *RPCBus) Register(methodName string, req chan<- Req) error {

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if req == nil {
		return ErrInvalidReqChan
	}

	if _, ok := bus.registry[methodName]; ok {
		return ErrMethodExists
	}

	bus.registry[methodName] = method{Name: methodName, req: req}
	return nil
}

// Call runs a long-polling technique to request from the method Consumer to
// run the corresponding procedure and return a result or timeout
func (bus *RPCBus) Call(methodName string, req Req) (bytes.Buffer, error) {

	var resp bytes.Buffer
	method, err := bus.getMethod(methodName)

	if err != nil {
		return bytes.Buffer{}, err
	}

	// Send the request with write-timeout
	select {
	case method.req <- req:
	case <-time.After(time.Duration(req.Timeout) * time.Second):
		return bytes.Buffer{}, ErrReqTimeout
	}

	// Wait for response or err from the consumer with read-timeout
	select {
	case resp = <-req.RespChan:
	// this case happens when the consumer cannot return a valid response but an
	// error details instead
	case err := <-req.ErrChan:
		return bytes.Buffer{}, err
	// terminate the procedure if timeout-ed
	case <-time.After(time.Duration(req.Timeout) * time.Second):
		err = ErrReqTimeout
	}

	return resp, err
}

// NewRequest builds a new request with params
func NewRequest(p bytes.Buffer, timeout int) Req {
	d := Req{Timeout: timeout, Params: p}
	d.RespChan = make(chan bytes.Buffer)
	d.ErrChan = make(chan error)
	return d
}

func (bus *RPCBus) getMethod(methodName string) (method, error) {

	// Guards the bus.registry until we find and return a copy
	bus.mu.RLock()
	defer bus.mu.RUnlock()

	if method, ok := bus.registry[methodName]; ok {
		return method, nil
	}

	return method{}, ErrMethodNotExists
}

// Close all open channels
func (bus *RPCBus) Close() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	for _, m := range bus.registry {
		close(m.req)
	}

	bus.registry = nil
}
