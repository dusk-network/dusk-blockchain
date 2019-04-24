package wire

import (
	"bytes"
	"testing"
	"time"
)

var expectedResult string

type consumer struct {
}

func (c consumer) loop(bus *RPCBus, delay int) {

	// wait for new requests
	for req := range GetLastBlockChan {

		// Simulate heavy computation
		time.Sleep(time.Duration(delay) * time.Millisecond)
		expectedResult = "Wrapped " + req.Params.String()

		buf := bytes.Buffer{}
		buf.WriteString(expectedResult)

		// return result
		req.Resp <- buf
	}

}

func newConsumer(t *testing.T, bus *RPCBus, delay int) consumer {

	c := consumer{}
	go c.loop(bus, delay)

	time.Sleep(10 * time.Millisecond)
	return c
}
func TestRPCall(t *testing.T) {

	bus := NewRPCBus()
	defer bus.Close()

	newConsumer(t, bus, 500)

	// produce the call
	buf := bytes.Buffer{}
	buf.WriteString("input params")

	d := NewRequest(buf, 2)
	responseResult, err := bus.Call(GetLastBlock, d)

	if err != nil {
		t.Error(err.Error())
	}

	if responseResult.String() != expectedResult {
		t.Errorf("expecting to retrieve response data")
	}
}

func TestTimeoutCalls(t *testing.T) {

	bus := NewRPCBus()

	delay := 3000
	newConsumer(t, bus, delay)

	// produce the call
	buf := bytes.Buffer{}
	buf.WriteString("input params")

	d := NewRequest(buf, 1)
	responseResult, err := bus.Call(GetLastBlock, d)

	if responseResult.Len() > 0 {
		t.Error("expecting empty result")
	}

	if err != ErrReqTimeout {
		t.Error("expecting timeout error")
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

	newConsumer(t, bus, 500)

	// produce the call
	buf := bytes.Buffer{}
	buf.WriteString("input params")

	d := NewRequest(buf, 2)
	responseResult, err := bus.Call("Chain/NonExistingMethod", d)

	if responseResult.Len() > 0 {
		t.Error("expecting empty result")
	}

	if err != ErrMethodNotExists {
		t.Error("expecting methodNotExists error")
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
