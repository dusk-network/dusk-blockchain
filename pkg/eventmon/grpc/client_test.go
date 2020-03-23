// This package is for testing the grpc client calls to the hello service
package grpc_test

import (
	"net"
	"testing"
	"time"

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
