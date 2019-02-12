package main

import (
	"fmt"
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

	fmt.Println(port, peers)

	// connect to the peers in the list
	for {
	}
}
