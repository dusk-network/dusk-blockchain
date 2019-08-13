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
		Prints the balance of the loaded wallet. Make sure to use the sync command first.`,
	"transfer": `Usage: transfer [amount] [address] [password]
		Send DUSK to a given address. Make sure to use the sync command first.`,
	"stake": `Usage: stake [amount] [locktime] [password]
		Stake a given amount of DUSK, to enter the provisioner committee. If you wish to participate in consensus, make sure to use the startprovisioner command afterwards.`,
	"bid": `Usage: bid [amount] [locktime] [password]
		Bid a given amount of DUSK, to allow participation as a block generator in consensus. To start generating blocks, use the startblockgenerator command, supplying the hash returned by the bid command. Note that the transaction needs to be included in a block before this will work.`,
	"startprovisioner": `Turns on the provisioner components in the node, needed to perform consensus. If you have not staked, the node will only relay consensus messages, until you stake any amount of DUSK.`,
	"startblockgenerator": `Usage: startblockgenerator [bidtxhash]
		Send a signal to the connected DUSK node to start participating in consensus as a block generator. Specified bid tx must be included in a block before trying to start the block generation component.`,
	"showlogs":  "Close the shell and show the internal logs on the terminal. Press enter to return to the shell.",
	"exit/quit": `Shut down the node and close the console`,
}
