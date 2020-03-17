// This package is for testing the grpc client calls to the hello service
package grpc_test

import (
	"context"
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
	requestChan chan *monitor.SemverRequest
}

func newSrv(network, addr string) <-chan *monitor.SemverRequest {
	l, err := net.Listen(network, addr)
	if err != nil {
		panic(err)
	}

	semverChan := make(chan *monitor.SemverRequest, 1)
	grpcServer := g.NewServer()
	monitor.RegisterMonitorServer(grpcServer, &helloSrv{semverChan})
	// This function is blocking, so we run it in a goroutine
	go grpcServer.Serve(l)
	return semverChan
}

func (h *helloSrv) Hello(ctx context.Context, req *monitor.SemverRequest) (*monitor.EmptyResponse, error) {
	h.requestChan <- req
	return &monitor.EmptyResponse{}, nil
}

func TestHello(t *testing.T) {
	semverChan := newSrv("tcp", ":7878")
	time.Sleep(10 * time.Millisecond)

	client := grpc.New("", "7878")
	if !assert.NoError(t, client.Hello()) {
		t.FailNow()
	}

	ver := <-semverChan
	assert.Equal(t, uint32(protocol.NodeVer.Major), ver.Major)
	assert.Equal(t, uint32(protocol.NodeVer.Minor), ver.Minor)
	assert.Equal(t, uint32(protocol.NodeVer.Patch), ver.Patch)
}
