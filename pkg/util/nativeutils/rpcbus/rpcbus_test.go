package rpcbus

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/topics"
)

const m = topics.GetLastBlock

var errInvalidParams = errors.New("invalid params")

func TestRPCall(t *testing.T) {
	bus := New()
	go setupConsumer(bus, true)
	time.Sleep(100 * time.Millisecond)

	// produce the call
	buf := bytes.Buffer{}
	_, _ = buf.WriteString("input params")

	d := NewRequest(buf)
	resp, err := bus.Call(m, d, 10*time.Second)
	if err != nil {
		t.Error(err.Error())
	}
	result := resp.(bytes.Buffer)

	if result.String() != "output params" {
		t.Errorf("expecting to retrieve response data")
	}
}

func TestRPCallWithError(t *testing.T) {
	bus := New()
	go setupConsumer(bus, true)
	time.Sleep(100 * time.Millisecond)

	// produce the call
	buf := bytes.Buffer{}
	_, _ = buf.WriteString("")

	d := NewRequest(buf)
	resp, err := bus.Call(m, d, 10*time.Second)
	if err != errInvalidParams {
		t.Errorf("expecting a specific error here but get %v", err)
	}
	result := resp.(bytes.Buffer)

	if result.String() != "" {
		t.Errorf("expecting to empty response data")
	}
}

func TestTimeoutCalls(t *testing.T) {
	bus := New()
	go setupConsumer(bus, false)
	time.Sleep(100 * time.Millisecond)

	// produce the call
	buf := bytes.Buffer{}
	_, _ = buf.WriteString("input params")

	d := NewRequest(buf)
	resp, err := bus.Call(m, d, 1*time.Second)
	if err != ErrRequestTimeout {
		t.Errorf("expecting timeout error but get %v", err)
	}
	result := resp.(bytes.Buffer)

	if result.Len() > 0 {
		t.Error("expecting empty result")
	}
}

func TestMethodExists(t *testing.T) {
	bus := New()
	go setupConsumer(bus, true)
	time.Sleep(100 * time.Millisecond)

	reqChan2 := make(chan Request)
	err := bus.Register(m, reqChan2)

	if err != ErrMethodExists {
		t.Fatalf("expecting methodExists error but get %v", err)
	}
}

func TestNonExistingMethod(t *testing.T) {
	bus := New()
	go setupConsumer(bus, true)
	time.Sleep(100 * time.Millisecond)

	// produce the call
	buf := bytes.Buffer{}
	_, _ = buf.WriteString("input params")

	d := NewRequest(buf)
	resp, err := bus.Call(0xff, d, 2*time.Second)
	if err != ErrMethodNotExists {
		t.Error("expecting methodNotExists error")
	}
	responseResult := resp.(bytes.Buffer)

	if responseResult.Len() > 0 {
		t.Error("expecting empty result")
	}
}

func TestInvalidReqChan(t *testing.T) {
	bus := New()

	err := bus.Register(m, nil)
	if err != ErrInvalidRequestChan {
		t.Error("expecting ErrInvalidReqChan error")
	}
}

func setupConsumer(rpcBus *RPCBus, respond bool) {
	reqChan := make(chan Request, 1)
	rpcBus.Register(m, reqChan)

	if respond {
		r := <-reqChan
		params := r.Params.(bytes.Buffer)
		if params.Len() == 0 {
			r.RespChan <- Response{bytes.Buffer{}, errInvalidParams}
			return
		}

		r.RespChan <- Response{*bytes.NewBufferString("output params"), nil}
	}
}
