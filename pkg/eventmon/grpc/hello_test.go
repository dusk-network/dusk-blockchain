// This package is for testing the grpc client calls to the hello service
package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/grpc"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	"github.com/stretchr/testify/assert"
)

func (h *helloSrv) Hello(ctx context.Context, req *monitor.SemverRequest) (*monitor.EmptyResponse, error) {
	h.requestChan <- req
	return &monitor.EmptyResponse{}, nil
}

func (h *helloSrv) Bye(ctx context.Context, req *monitor.EmptyRequest) (*monitor.EmptyResponse, error) {
	h.requestChan <- struct{}{}
	return &monitor.EmptyResponse{}, nil
}

func TestHello(t *testing.T) {

	client := grpc.New("", "7878")
	method := client.Hello
	tester := func(response interface{}) error {
		ver, ok := response.(*monitor.SemverRequest)
		if !ok {
			return fmt.Errorf("unexpected request type %v", ver)
		}

		if !assert.Equal(t, uint32(protocol.NodeVer.Major), ver.Major) {
			return errors.New("unexpected major version")
		}
		if !assert.Equal(t, uint32(protocol.NodeVer.Minor), ver.Minor) {
			return errors.New("unexpected minor version")
		}
		if !assert.Equal(t, uint32(protocol.NodeVer.Patch), ver.Patch) {
			return errors.New("unexpected patch version")
		}
		return nil
	}

	Suite(t, method, tester)
}

func TestBye(t *testing.T) {
	client := grpc.New("", "7878")
	Suite(t, client.Bye, emptyFunc)
}
