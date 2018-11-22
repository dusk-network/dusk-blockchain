package commands

// Cmd defines a command
type Cmd string

// Size is the size of a command field in bytes
const Size = 12

// A list of all valid protocol commands
const (
	Version Cmd = "version"
	VerAck  Cmd = "verack"
	Ping    Cmd = "ping"
	Pong    Cmd = "pong"
	Addr    Cmd = "addr"
	GetAddr Cmd = "getaddr"
	TX      Cmd = "tx"
)
