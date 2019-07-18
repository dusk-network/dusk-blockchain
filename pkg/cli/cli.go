package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// Start the interactive shell.
func Start(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, logFile *os.File) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ")
		if fn := CLICommands[args[0]]; fn != nil {
			fn(args[1:], eventBroker, rpcBus)
		} else if args[0] == "showlogs" {
			showLogs(logFile)
		} else {
			fmt.Printf("%v is not a supported command\n", args[0])
		}

		fmt.Print("> ")
	}

	if scanner.Err() != nil {
		log.WithFields(log.Fields{
			"process": "cli",
			"error":   scanner.Err(),
		}).Errorln("cli error")
	}
}

func showLogs(logFile *os.File) {
	fmt.Fprintln(os.Stdout, "Logging to the terminal - press enter to stop and restart the shell")

	// Swap logrus output to a multiwriter, writing both to the file and to os.Stdout
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// Hacky way to capture enter
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}

	// Set output back to log file established at startup
	log.SetOutput(logFile)

	fmt.Fprintln(os.Stdout, "\nrestarting shell...")
}
