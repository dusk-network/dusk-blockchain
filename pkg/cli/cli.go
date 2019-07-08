package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire"
)

// Start the interactive shell.
func Start(publisher wire.EventPublisher) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
	for scanner.Scan() {
		args := strings.Split(scanner.Text(), " ")
		if fn := CLICommands[args[0]]; fn != nil {
			fn(args[1:], publisher)
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
