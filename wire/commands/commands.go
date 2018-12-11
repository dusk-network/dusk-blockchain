package commands

// Cmd defines a command
type Cmd string

// Size is the size of a command field in bytes
const Size = 12

// A list of all valid protocol commands
const (
	Version    Cmd = "version"
	VerAck     Cmd = "verack"
	Ping       Cmd = "ping"
	Pong       Cmd = "pong"
	Addr       Cmd = "addr"
	GetAddr    Cmd = "getaddr"
	GetData    Cmd = "getdata"
	GetBlocks  Cmd = "getblocks"
	GetHeaders Cmd = "getheaders"
	Tx         Cmd = "tx"
	Block      Cmd = "block"
	Headers    Cmd = "headers"
	MemPool    Cmd = "mempool"
	Inv        Cmd = "inv"
	NotFound   Cmd = "notfound"
	Reject     Cmd = "reject"
)

// CmdToByteArray turns a Cmd to a byte array of size 12,
// to prepare it for sending over the wire protocol.
func CmdToByteArray(cmd Cmd) [Size]byte {
	bs := [Size]byte{}
	for i := 0; i < len(cmd); i++ {
		bs[i] = cmd[i]
	}

	return bs
}

// ByteArrayToCmd turns a byte array of size 12 into a Cmd,
// for populating a received message header.
func ByteArrayToCmd(cmd [Size]byte) Cmd {
	buf := []byte{}
	for i := 0; i < Size; i++ {
		if cmd[i] != 0 {
			buf = append(buf, cmd[i])
		}
	}

	return Cmd(buf)
}
