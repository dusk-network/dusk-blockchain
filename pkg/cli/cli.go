package cli

import (
	"bufio"
	"os"
	"strings"
)

// start the interactive wallet shell
func startCLI() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		args := strings.Split(scanner.Text())
		if fn := CLICommands[args[0]]; fn != nil {
			fn(args[1:])
		}
	}

	if scanner.Err() {
		//
	}
}
