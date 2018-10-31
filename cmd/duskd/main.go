// Dusk Daemon

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/toghrulmaharramov/dusk-go/rpc"
)

var conf = flag.String("conf", "dusk.conf",
	"Specify alternative config file to use (default: dusk.conf)")
var rpcport = flag.String("rpcport", "9999",
	"Specify RPC port to bind server on (default: loaded from config")
var rpcuser = flag.String("rpcuser", "dusk123",
	"Specify RPC username (default: loaded from config)")
var rpcpass = flag.String("rpcpass", "duskpass",
	"Specify RPC password (default: loaded from config)")
var version = flag.Bool("version", false, "Show version and exit")

func main() {
	flag.Parse()
	if *version {
		// In the future, make this actually retrieve version from the software
		fmt.Println("0.1")
		return
	}

	// Load config and start RPC server
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	srv, err := rpc.NewRPCServer(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// Listen for RPC stopnode command
	rpcQuit := srv.StopChan

	if err := srv.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// Listen for CTRL+C
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT)

	select {
	case <-quit:
		if err := srv.Stop(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	case <-rpcQuit:
		break // Server has already quit on it's own
	}

	// Once we set up the database and P2P networking, close them here.

	return
}

// LoadConfig will load a config file into a Config struct and return it.
func LoadConfig() (*rpc.Config, error) {
	// Initialize config object
	cfg := rpc.Config{}

	// Set filepath if specified
	if *conf != "dusk.conf" {
		cfg.File = *conf
	}

	// Create config file if it doesn't exist yet
	if err := cfg.Load(); err != nil {
		return nil, err
	}

	// Override config options with flags if specified
	if *rpcuser != "dusk123" {
		cfg.RPCUser = *rpcuser
	}
	if *rpcpass != "duskpass" {
		cfg.RPCPass = *rpcpass
	}
	if *rpcport != "9999" {
		cfg.RPCPort = *rpcport
	}

	return &cfg, nil
}
