package cli

var commandInfo = map[string]string{
	"help": `Usage: help [command] 
		Show this menu. When adding a command, shows info about that command.`,
	"createwallet": `Usage: createwallet [password]
		Create an encrypted wallet file.`,
	"loadwallet": `Usage: loadwallet [password]
		Loads the encrypted wallet file.`,
	"createfromseed": `Usage: createfromseed [seed] [password]
		Loads the encrypted wallet file from a hex seed.`,
	"balance": `Usage: balance
		Prints the balance of the loaded wallet.`,
	"transfer": `Usage: transfer [amount] [address]
		Send DUSK to a given address.`,
	"stake": `Usage: stake 
		Stake a set amount of DUSK, to allow participation as a provisioner in consensus. The amount is determined by the value specified with the 'setdefaultvalue' command, or the config if none was specified.`,
	"bid": `Usage: bid 
		Bid a set amount of DUSK, to allow participation as a block generator in consensus. The amount is determined by the value specified with the 'setdefaultvalue' command, or the config if none was specified.`,
	"setdefaultlocktime": `Usage: setdefaultlocktime [locktime]
		Sets the locktime for consensus transactions.`,
	"setdefaultvalue": `Usage: setdefaultvalue [amount]
		Sets the default value for consensus transactions.`,
	"showlogs":  "Close the shell and show the internal logs on the terminal. Press enter to return to the shell.",
	"exit/quit": `Shut down the node and close the console`,
}
