// Dusk Daemon

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/toghrulmaharramov/dusk-go/rpc"
)

func main() {
	conf := rpc.Config{
		RPCUser:     "dusk123",
		RPCPassword: "duskpass",
		RPCPort:     "9999",
	}

	srv, err := rpc.NewRPCServer(&conf)
	if err != nil {
		return
	}

	// Listen for CTRL+C
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT)

	// Listen for RPC stopnode command
	rpcQuit := srv.StopChan

	srv.Start()

	select {
	case <-quit:
		srv.Stop()
	case <-rpcQuit:
		break // Server has already quit on it's own
	}

	// Once we set up the database and P2P networking, close them here.

	return
}
