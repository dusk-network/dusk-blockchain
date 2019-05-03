package wire

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

var expectedResult string
var consumerStarted bool
var errInvalidParams = errors.New("invalid params")

func runConsumer(delay int) {
	if consumerStarted == false {
		consumerStarted = true
		go func(delay int) {
			for req := range GetLastBlockChan {
				// Simulate heavy computation
				time.Sleep(time.Duration(delay) * time.Millisecond)

				params := req.Params.String()

				if len(params) == 0 {
					// Simulate error response
					req.Err <- errInvalidParams
				} else {
					// Simulate non-error response
					expectedResult = "Wrapped " + params

					buf := bytes.Buffer{}
					buf.WriteString(expectedResult)

					// return result
					req.Resp <- buf
				}
			}
		}(delay)
	}
}
func TestRPCall(t *testing.T) {

	bus := NewRPCBus()
	defer bus.Close()

	runConsumer(500)

	// produce the call
	buf := bytes.Buffer{}
	buf.WriteString("input params")

	d := NewRequest(buf, 10)
	responseResult, err := bus.Call(GetLastBlock, d)

	if err != nil {
		t.Error(err.Error())
	}

	if responseResult.String() != expectedResult {
		t.Errorf("expecting to retrieve response data")
	}
}

func TestRPCallWithError(t *testing.T) {

	bus := NewRPCBus()
	defer bus.Close()

	runConsumer(500)

	// produce the call
	buf := bytes.Buffer{}
	buf.WriteString("")

	d := NewRequest(buf, 10)
	responseResult, err := bus.Call(GetLastBlock, d)

	if err != errInvalidParams {
		t.Errorf("expecting a specific error here %v", err)
	}

	if responseResult.String() != "" {
		t.Errorf("expecting to empty response data")
	}
}

func TestTimeoutCalls(t *testing.T) {

	bus := NewRPCBus()
	defer bus.Close()

	delay := 3000
	runConsumer(delay)

	// produce the call
	buf := bytes.Buffer{}
	buf.WriteString("input params")

	d := NewRequest(buf, 1)
	responseResult, err := bus.Call(GetLastBlock, d)

	if err != ErrReqTimeout {
		t.Error("expecting timeout error")
	}

	if responseResult.Len() > 0 {
		t.Error("expecting empty result")
	}
}

func TestMethodExists(t *testing.T) {

	bus := NewRPCBus()
	defer bus.Close()

	reqChan2 := make(chan Req)
	err := bus.Register(GetLastBlock, reqChan2)

	if err != ErrMethodExists {
		t.Fatalf("expecting methodExists error")
	}
}

func TestNonExistingMethod(t *testing.T) {

	bus := NewRPCBus()
	defer bus.Close()

	runConsumer(500)

	// produce the call
	buf := bytes.Buffer{}
	buf.WriteString("input params")

	d := NewRequest(buf, 2)
	responseResult, err := bus.Call("Chain/NonExistingMethod", d)

	if err != ErrMethodNotExists {
		t.Error("expecting methodNotExists error")
	}

	if responseResult.Len() > 0 {
		t.Error("expecting empty result")
	}
}

func TestInvalidReqChan(t *testing.T) {
	bus := NewRPCBus()
	defer bus.Close()

	err := bus.Register(GetLastBlock, nil)
	if err != ErrInvalidReqChan {
		t.Error("expecting ErrInvalidReqChan error")
	}
}
