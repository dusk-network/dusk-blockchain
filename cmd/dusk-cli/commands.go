package main

// Command defines a CLI command and holds it's explanation and required parameters.
type Command struct {
	Help   string
	Params []string
}

// cmdMap is a mapping of CLI command method names and their corresponding structs.
var cmdMap = make(map[string]Command)

var versionCmd = Command{
	Help: "Print node version.",
}

var exitCmd = Command{
	Help: "Exit this program. Exiting this program does not mean that the node will shut down.",
}

var stopNodeCmd = Command{
	Help: "Shuts down the node and exits this program.",
}

var pingCmd = Command{
	Help: "Ping RPC server to verify that it's up",
}

func init() {
	cmdMap["version"] = versionCmd
	cmdMap["exit"] = exitCmd
	cmdMap["stopnode"] = stopNodeCmd
	cmdMap["ping"] = pingCmd
}

// HandleCommand takes the passed method and parameters, marshals them into a JSON
// request and sends a POST request to the RPC server.
func HandleCommand(method string, params []string, cfg *Config) (*Response, error) {
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
	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp, nil
}
