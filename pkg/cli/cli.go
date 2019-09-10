package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dusk-network/dusk-blockchain/pkg/p2p/wire"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/logging"
	log "github.com/sirupsen/logrus"
)

// Start the interactive shell.
func Start(eventBroker wire.EventBroker, rpcBus *wire.RPCBus, logFile *os.File) {

	// TODO: Remove when dusk-blockchain/issues/27 is done
	time.Sleep(2 * time.Second)
	CLICommands["loadwallet"]([]string{"password"}, eventBroker, rpcBus)
	time.Sleep(2 * time.Second)
	CLICommands["startblockgenerator"]([]string{""}, eventBroker, rpcBus)
	time.Sleep(2 * time.Second)
	CLICommands["startprovisioner"]([]string{""}, eventBroker, rpcBus)

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ")
		if fn := CLICommands[args[0]]; fn != nil {
			fn(args[1:], eventBroker, rpcBus)
		} else if args[0] == "showlogs" {
			showLogs(args[1:], logFile)
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

func showLogs(args []string, logFile *os.File) {
	if len(args) > 0 {
		logging.SetToLevel(args[0])
	}

	fmt.Fprintln(os.Stdout, "Logging to the terminal - press enter to stop and restart the shell")

	// Swap logrus output to a multiwriter, writing both to the file and to os.Stdout
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// Hacky way to capture enter
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}

	// Set output and level back to defaults
	logging.InitLog(logFile)

	fmt.Fprintln(os.Stdout, "\nrestarting shell...")
}
