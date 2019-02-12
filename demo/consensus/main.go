package main

import (
	"fmt"
	"net"
	"os"
)

func main() {

	if len(os.Args) < 2 { // [programName, port,...]
		fmt.Println("Please enter more arguments")
		os.Exit(1)
	}

	args := os.Args[1:] // [port, peers...]

	port := args[0]

	var peers []string
	if len(args) > 1 {
		peers = args[1:]
	}

	cfg := CmgrConfig{
		Port:     port,
		OnAccept: OnAccept,
		OnConn:   OnConnection,
	}

	cmgr := newConnMgr(cfg)
	for _, peer := range peers {
		err := cmgr.Connect(peer)
		if err != nil {
			fmt.Println(err)
		}
	}
	// connect to the peers in the list
	for {
	}
}

func OnAccept(conn net.Conn) {
	fmt.Printf("someone has tried to connect to us, with the address %s \n", conn.RemoteAddr().String())
}

func OnConnection(conn net.Conn, addr string) {
	fmt.Printf("we have connected to the node with the address %s \n", conn.RemoteAddr().String())

}
