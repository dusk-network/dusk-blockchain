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

var (

	// Default set of registered methods

	// Provide the last/highest block from the local chain state
	// Can be implemented by Chain pkg or Database pkg.
	// Returns block.Block marshaled
	GetLastBlock     = "getLastBlock"
	GetLastBlockChan chan Request

	// Provide the list of verified txs ready to be included into next block
	// Param 1: list of TxIDs to request. If empty, returns all available txs
	// Implemented by mempool
	GetMempoolTxs     = "getMempoolTxs"
	GetMempoolTxsChan chan Request

	// Verify a specified candidate block
	//
	// Used by the reduction component.
	VerifyCandidateBlock     = "verifyCandidateBlock"
	VerifyCandidateBlockChan chan Request

	// Methods implemented by Transactor
	CreateWallet   = "createWallet"
	CreateFromSeed = "createFromSeed"
	LoadWallet     = "loadWallet"
	SendBidTx      = "sendBidTx"
	SendStakeTx    = "sendStakeTx"
	SendStandardTx = "sendStandardTxChan"
	GetBalance     = "getBalance"
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
	req  chan<- Request
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
	var bus RPCBus
	bus.registry = make(map[string]method)

	// default methods
	GetLastBlockChan = make(chan Request)
	if err := bus.Register(GetLastBlock, GetLastBlockChan); err != nil {
		panic(err)
	}

	GetMempoolTxsChan = make(chan Request)
	if err := bus.Register(GetMempoolTxs, GetMempoolTxsChan); err != nil {
		panic(err)
	}

	VerifyCandidateBlockChan = make(chan Request)
	if err := bus.Register(VerifyCandidateBlock, VerifyCandidateBlockChan); err != nil {
		panic(err)
	}

	return &bus
}

// Register registers a method and binds it to a handler channel. methodName
// must be unique per node instance. if not, returns err
func (bus *RPCBus) Register(methodName string, req chan<- Request) error {

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if req == nil {
		return ErrInvalidRequestChan
	}

	if _, ok := bus.registry[methodName]; ok {
		return ErrMethodExists
	}

	bus.registry[methodName] = method{Name: methodName, req: req}
	return nil
}

// Call runs a long-polling technique to request from the method Consumer to
// run the corresponding procedure and return a result or timeout
func (bus *RPCBus) Call(methodName string, req Request) (bytes.Buffer, error) {

	if req.Timeout > 0 {
		return bus.callTimeout(methodName, req)
	}

	return bus.callNoTimeout(methodName, req)
}

func (bus *RPCBus) callTimeout(methodName string, req Request) (bytes.Buffer, error) {
	method, err := bus.getMethod(methodName)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Send the request with write-timeout
	select {
	case method.req <- req:
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

func (bus *RPCBus) callNoTimeout(methodName string, req Request) (bytes.Buffer, error) {
	method, err := bus.getMethod(methodName)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Send the request with write-timeout
	method.req <- req

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
