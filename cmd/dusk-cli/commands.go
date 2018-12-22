package main

import (
	"errors"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/rpc"
)

// Command defines a CLI command and holds it's properties.
type Command struct {
	Method string
	Help   string
	Params []string
}

// cmdMap is a mapping of CLI command method names and their corresponding structs.
var cmdMap = make(map[string]Command)

// commands is an array of all the commands available to call in the CLI.
var commands = []Command{
	Command{
		Method: "version",
		Help:   "Print node version.",
	},

	Command{
		Method: "exit",
		Help:   "Exit this program. Exiting this program does not mean that the node will shut down.",
	},

	Command{
		Method: "stopnode",
		Help:   "Shuts down the node and exits this program.",
	},

	Command{
		Method: "ping",
		Help:   "Ping RPC server to verify that it's up",
	},

	Command{
		Method: "uptime",
		Help:   "Show RPC server uptime in seconds",
	},

	// This command is mainly here for testing parameter passing. It can be removed.
	Command{
		Method: "hash",
		Help:   "Return the hash for each string passed",
		Params: []string{"text..."},
	},
}

func init() {
	for _, cmd := range commands {
		cmdMap[cmd.Method] = cmd
	}
}

// HandleCommand takes the passed method and parameters, marshals them into a JSON
// request and sends a POST request to the RPC server.
func HandleCommand(method string, params []string, cfg *rpc.Config) (*rpc.JSONResponse, error) {
	// Marshal passed method and params
	msg, err := MarshalCmd(method, params)
	if err != nil {
		return nil, err
	}

	// Send command
	resp, err := SendPostRequest(msg, cfg)
	if err != nil {
		return nil, err
	}

	// Handle result
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	return resp, nil
}

// ShowCommands iterates over the commands array and creates a string with all commands
// and their descriptions to be printed to the terminal.
func ShowCommands() string {
	var res string
	res += "Dusk CLI Commands:\n\n"
	for _, cmd := range commands {
		res += cmd.Method + "\n"
		res += "	" + cmd.Help + "\n"
		if len(cmd.Params) > 0 {
			var params string
			for _, p := range cmd.Params {
				params += "[" + p + "]"
			}

			res += "	Params: " + params + "\n"
		}
	}

	return res
}
