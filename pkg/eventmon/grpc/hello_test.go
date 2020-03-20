// This package is for testing the grpc client calls to the hello service
package grpc_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/eventmon/grpc"
	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire/protocol"
	"github.com/dusk-network/dusk-protobuf/autogen/go/monitor"
	"github.com/stretchr/testify/assert"
	g "google.golang.org/grpc"
)

type helloSrv struct {
	requestChan chan interface{}
	srv         *g.Server
}

type clientMethod = func() error
type tester = func(interface{}) error

var emptyFunc = func(_ interface{}) error {
	return nil
}

func newSrv(network, addr string) *helloSrv {
	l, err := net.Listen(network, addr)
	if err != nil {
		panic(err)
	}

	semverChan := make(chan interface{}, 1)
	grpcServer := g.NewServer()
	hs := &helloSrv{semverChan, grpcServer}
	monitor.RegisterMonitorServer(grpcServer, hs)
	// This function is blocking, so we run it in a goroutine
	go grpcServer.Serve(l)
	return hs
}

func (h *helloSrv) Hello(ctx context.Context, req *monitor.SemverRequest) (*monitor.EmptyResponse, error) {
	h.requestChan <- req
	return &monitor.EmptyResponse{}, nil
}

func (h *helloSrv) Bye(ctx context.Context, req *monitor.EmptyRequest) (*monitor.EmptyResponse, error) {
	h.requestChan <- struct{}{}
	return &monitor.EmptyResponse{}, nil
}

func Suite(t *testing.T, callback clientMethod, test tester) {
	semverSrv := newSrv("tcp", ":7878")
	defer semverSrv.srv.Stop()
	time.Sleep(10 * time.Millisecond)

	if !assert.NoError(t, callback()) {
		t.FailNow()
	}

	response := <-semverSrv.requestChan
	if !assert.NoError(t, test(response)) {
		t.FailNow()
	}
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
