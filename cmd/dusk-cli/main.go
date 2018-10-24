// Dusk Daemon CLI

package main

import (
	"bufio"
	"errors"
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
-help, -h
	Show this help message.

-conf=<file>
	Specify alternative config file to use (default: dusk.conf)

-rpcport=<port>
	Specify RPC port to connect to (default: loaded from config)

-rpcuser=<user>
	Specify RPC username (default: loaded from config)

-rpcpassword=<pass>
	Specify RPC password (default: loaded from config)`

var commands = `Dusk CLI Commands:
version 
	Print node version

exit
	Exits this CLI program

stopnode
	Quits duskd, then exits CLI program`

func main() {
	// Read RPC port from config, pass to CheckDaemon
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config, %v", err)
		return
	}

	// TODO: Parse flags, load new config file (if specified), then overwrite loaded config options

	// See if duskd is running
	err = CheckDaemon(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "duskd does not appear to be running on port %v, exiting...", cfg.RPCPort)
		return
	}

	// First off, check if the program was ran with any arguments. If so, just run the specified
	// command and exit like a standard cli utility.
	if len(os.Args) > 0 {
		if os.Args[0] == "help" {
			if len(os.Args) > 1 {
				// TODO: Log information about specified command
				return
			}
			fmt.Println(commands)
			return
		}

		// TODO: Handle command
		return
	}

	// If ran without a command, open up terminal interface and start taking commands.
	// Additionally, log the overview.
	fmt.Println(overview)
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
			if params[0] != "" {
				// TODO: Log information about specified command
				continue
			}
			fmt.Println(commands)
			continue
		}

		// TODO: Handle command
	}

	if err := s.Err(); err != nil {
		fmt.Fprint(os.Stderr, err)
	}
}

// CheckDaemon checks if there is a process running on the specified port from config. Calls the
// 'ping' RPC command to verify that node is online.
func CheckDaemon(cfg *Config) error {
	// TODO: Marshal ping command
	var cmd []byte
	result, err := SendPostRequest(cmd, cfg)
	if err != nil {
		return err
	}

	// TODO: Check response

	// Check placeholder
	if string(result) == "pong" {
		return nil
	}

	return errors.New("unexpected JSON response")
}
