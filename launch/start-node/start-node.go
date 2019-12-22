package main

import (
	"fmt"
	"os"
	"os/exec"
)

// A very simple binary, tasked with starting the processes for both
// the blind bid and the node.
func main() {
	blindBidCmd := exec.Command("./blindbid")
	if err := blindBidCmd.Start(); err != nil {
		fmt.Printf("could not start blind bid binary - %v\n", err)
		os.Exit(1)
	}

	nodeCmd := exec.Command("./testnet")
	if err := nodeCmd.Start(); err != nil {
		fmt.Printf("could not start node binary - %v\n", err)
		os.Exit(1)
	}
}
