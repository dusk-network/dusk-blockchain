package wire

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

var (
	expectedResult   string
	consumerStarted  bool
	errInvalidParams = errors.New("invalid params")

	cleanup = func() {
		GetLastBlockChan = nil
		GetMempoolTxsChan = nil
	}
)

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
					req.ErrChan <- errInvalidParams
				} else {
					// Simulate non-error response
					expectedResult = "Wrapped " + params

					buf := bytes.Buffer{}
					buf.WriteString(expectedResult)

					// return result
					req.RespChan <- buf
				}
			}
		}(delay)
	}
}
func TestRPCall(t *testing.T) {

	cleanup()
	bus := NewRPCBus()

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

	t.SkipNow()
	cleanup()
	bus := NewRPCBus()

	runConsumer(500)

	// produce the call
	buf := bytes.Buffer{}
	buf.WriteString("")

	d := NewRequest(buf, 10)
	responseResult, err := bus.Call(GetLastBlock, d)

	if err != errInvalidParams {
		t.Errorf("expecting a specific error here but get %v", err)
	}

	if responseResult.String() != "" {
		t.Errorf("expecting to empty response data")
	}
}

func TestTimeoutCalls(t *testing.T) {

	cleanup()
	bus := NewRPCBus()

	delay := 3000
	runConsumer(delay)

	// produce the call
	buf := bytes.Buffer{}
	buf.WriteString("input params")

	d := NewRequest(buf, 1)
	responseResult, err := bus.Call(GetLastBlock, d)

	if err != ErrReqTimeout {
		t.Errorf("expecting timeout error but get %v", err)
	}

	if responseResult.Len() > 0 {
		t.Error("expecting empty result")
	}
}

func TestMethodExists(t *testing.T) {
	cleanup()
	bus := NewRPCBus()

	reqChan2 := make(chan Req)
	err := bus.Register(GetLastBlock, reqChan2)

	if err != ErrMethodExists {
		t.Fatalf("expecting methodExists error but get %v", err)
	}
}

func TestNonExistingMethod(t *testing.T) {
	cleanup()
	bus := NewRPCBus()

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
	cleanup()
	bus := NewRPCBus()

	err := bus.Register(GetLastBlock, nil)
	if err != ErrInvalidReqChan {
		t.Error("expecting ErrInvalidReqChan error")
	}
}
