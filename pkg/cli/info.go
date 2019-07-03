package cli

var commandInfo = map[string]string{
	"help": `Show this menu`,
	"createwallet": `Usage: createwallet [name] [password]
		Create an encrypted wallet file.`,
	"transfer": `Usage: transfer [amount] [address] [fee]
		Send DUSK to a given address. Fee is optional, will use standard value if not specified.`,
	"stake": `Usage: stake [amount] [fee]
		Stake a given amount of DUSK, to allow participation as a provisioner in consensus. Fee is optional, will use standard value if not specified.`,
	"bid": `Usage: bid [amount] [fee]
		Bid a given amount of DUSK, to allow participation as a block generator in consensus. Fee is optional, will use standard value if not specified.`,
	"startconsensus": `Send a signal to the connected DUSK node to start participating in consensus.`,
	"exit/quit":      `Shut down the node and close the console`,
}
