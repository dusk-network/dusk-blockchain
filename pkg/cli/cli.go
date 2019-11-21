package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/logging"
	"github.com/dusk-network/dusk-blockchain/pkg/util/nativeutils/rpcbus"
	log "github.com/sirupsen/logrus"
)

// Start the interactive shell.
func Start(rpcBus *rpcbus.RPCBus, logFile *os.File) {
	processor := &commandLineProcessor{rpcBus}
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ")
		processor.runCmd(args[0], args[1:], logFile)
		fmt.Print("> ")
	}

	if scanner.Err() != nil {
		log.WithFields(log.Fields{
			"process": "cli",
			"error":   scanner.Err(),
		}).Errorln("cli error")
	}
}

func (c *commandLineProcessor) runCmd(cmd string, args []string, logFile *os.File) {
	switch cmd {
	case "help":
		showHelp(args)
	case "createwallet":
		c.createWalletCMD(args)
	case "loadwallet":
		c.loadWalletCMD(args)
	case "createfromseed":
		c.createFromSeedCMD(args)
	case "balance":
		c.balanceCMD()
	case "transfer":
		c.transferCMD(args)
	case "stake":
		c.sendStakeCMD(args)
	case "bid":
		c.sendBidCMD(args)
	case "exit", "quit":
		stopNode()
	case "showlogs":
		showLogs(args, logFile)
	case "address":
		c.addressCMD(args)
	default:
		fmt.Fprintf(os.Stdout, "command %s not recognized\n", cmd)
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
