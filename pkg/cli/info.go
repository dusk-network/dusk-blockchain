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
	"sync": "Sync the wallet with the attached node blockchain."
	"balance": `Usage: balance
		Prints the balance of the loaded wallet. Make sure to use the sync command first.`,
	"transfer": `Usage: transfer [amount] [address] [password]
		Send DUSK to a given address. Make sure to use the sync command first.`,
	"stake": `Usage: stake [amount] [locktime] [password]
		Stake a given amount of DUSK, to allow participation as a provisioner in consensus. Make sure to use the sync command first.`,
	"bid": `Usage: bid [amount] [locktime] [password]
		Bid a given amount of DUSK, to allow participation as a block generator in consensus. Make sure to use the sync command first.`,
	"startprovisioner": `Send a signal to the connected DUSK node to start participating in consensus as a provisioner.`,
	"startblockgenerator": `Usage: startblockgenerator [bidtxhash]
		Send a signal to the connected DUSK node to start participating in consensus as a block generator`,
	"showlogs": "Close the shell and show the internal logs on the terminal. Press any key to return to the shell."
	"exit/quit": `Shut down the node and close the console`,
}
