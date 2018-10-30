// Dusk Daemon CLI

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

var overview = `Dusk Daemon Command Line Interface

Usage:
dusk-cli [options]                - Start CLI as terminal program
dusk-cli [options] help           - Show the command overview and exit
dusk-cli [options] <command>      - Run command and exit
dusk-cli [options] help <command> - Show information about command and exit

Options:
--help, -h
	Shows information about flags and exits.

-conf=<file>
	Specify alternative config file to use (default: dusk.conf)

-rpcport=<port>
	Specify RPC port to connect to (default: loaded from config)

-rpcuser=<user>
	Specify RPC username (default: loaded from config)

-rpcpass=<pass>
	Specify RPC password (default: loaded from config)`

var conf = flag.String("conf", "dusk.conf",
	"Specify alternative config file to use (default: dusk.conf)")
var rpcport = flag.String("rpcport", "9999",
	"Specify RPC port to connect to (default: loaded from config")
var rpcuser = flag.String("rpcuser", "dusk123",
	"Specify RPC username (default: loaded from config)")
var rpcpass = flag.String("rpcpass", "duskpass",
	"Specify RPC password (default: loaded from config)")

func main() {
	// Load config, parse flags
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config - %v", err)
		return
	}

	// See if duskd is running by sending a simple 'ping' command
	if _, err := HandleCommand("ping", []string{}, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "duskd does not appear to be running on port %v: %v\n", cfg.RPCPort, err)
		return
	}

	args := flag.Args()

	// First off, check if the program was ran with any arguments. If so, just run the specified
	// command and exit like a standard cli utility.
	if len(args) > 1 {
		if args[1] == "help" {
			if len(args) > 2 {
				fmt.Println(cmdMap[args[2]].Help)
				return
			}
			fmt.Print(ShowCommands())
			return
		}

		method := args[1]
		params := args[2:]
		resp, err := HandleCommand(method, params, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error handling command %v: %v", method, err)
			return
		}

		fmt.Println(resp.Result)

		return
	}

	// If ran without a command, open up terminal interface and start taking commands.
	// Additionally, log the overview and commands before doing so.
	fmt.Println(overview)
	fmt.Println(ShowCommands())

	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		input := s.Text()
		args := strings.Fields(input)
		method := args[0]
		var params []string
		if len(args) > 1 {
			params = args[1:]
		}

		if method == "help" {
			if len(params) > 0 {
				fmt.Println(cmdMap[params[0]].Help)
				continue
			}
			fmt.Print(ShowCommands())
			continue
		}

		if method == "exit" {
			return
		}

		resp, err := HandleCommand(method, params, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error handling command %v: %v", method, err)
			continue
		}

		fmt.Println(resp.Result)

		if method == "stopnode" {
			return
		}
	}

	// If the scanner encounters an error, log to os.Stderr
	if err := s.Err(); err != nil {
		fmt.Fprint(os.Stderr, err)
	}
}
