package main

import (
	"fmt"
	"os"
)

func main() {

	if len(os.Args) < 3 { // [programName, port, peers...]
		fmt.Println("Please enter more arguments")
		os.Exit(1)
	}

	args := os.Args[1:] // [port, peers...]

	port := args[0]
	peers := args[1:]

	fmt.Println(port, peers)
	for {
	}
}
