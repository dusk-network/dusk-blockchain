### RPC server

The RPC server is loosely based on the RPC server used by the Bitcoin client. On startup, the server will listen on a specified port on localhost, and accept method calls through a JSON format. The server will respond with a JSON response including the result of the function called, and an error message if one was encountered during function execution. In case of an unauthenticated user calling an admin level command, the server will send back a 401 Unauthorized response, along with an error message specifying which command was called.

### CLI executable

The CLI executable is a standalone binary which should be compiled on build. It provides a simple terminal interface that takes commands and sends them over to the RPC server on the specified port. Specifications like the username, password and server port are read from the dusk.conf file in the same folder (which will be created if it doesn't exist) and are used to populate requests with authorization data and point the requests in the right direction. If username and password match between the user and the server, then the user is allowed to call admin commands as well as normal commands.

Usage of the CLI executable is fairly simple. You can run the executable from the command line like the bitcoin-cli executable. Flags are parsed on startup, which will overwrite any config settings, and if no methods are passed in the command line, the CLI will start scanning for terminal input. Flag information and available commands are printed on startup. The process will also check if duskd is actually running or not before giving you access or executing your command, so make sure both the RPC server and the CLI are looking on the same port (they will by default).

### Commands

Commands are designed to be implemented as easily as possible. To add a command, you will have to add it to the `commands` array in `commands.go` located in `dusk-cli`. The `init()` function will include the command into the `cmdMap` mapping on startup. Then, in folder `rpc`, file `commands.go`, you will have to add the method name (lowercase) as well as it's handler function to the `RPCCmd` mapping. If the command is intended for admin use only, include the method name and a `true` bool in `RPCAdminCmd` as well. Then below, you should declare the handler function and make it do what you wish. In the future, it will make sense to seperate these handlers based on several categories in multiple Go files, but for now this should work.

Example of a command in `dusk-cli/commands.go`:

```go
Command{
    Method: "hash",
    Help:   "Return the hash for each string passed",
    Params: []string{"text..."},
},
```

And an example of it's function counterpart in `rpc/commands.go`:

```go
var Hash = func(s *Server, params []string) (string, error) {
	var hashes string
	for _, word := range params {
		hash, err := hash.Sha3256([]byte(word))
		if err != nil {
			return "", err
		}

		text := base58.Encoding(hash)
		hashes += text + " "
	}

	return hashes, nil
}
```