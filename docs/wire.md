# Dusk wire protocol

Found in the `wire` folder, the Dusk wire protocol is (for the moment) based on the [Bitcoin wire protocol](https://en.bitcoin.it/wiki/Protocol_documentation), with the source code modeled after [Kev's NEO wire protocol implementation](https://github.com/decentralisedkev/neo-go/tree/v2/pkg/wire). The package defines different types of messages, and provides encoding/decoding methods for all of them, as well as generalised `WriteMessage` and `ReadMessage` functions. Tests are included for all completed source files, to ensure proper encoding and decoding of information and commands. As we progress in building the node, these source files will definitely be subject to change, but this should be sufficient to set up a rudimentary wire protocol within the devnet.

## Block related commands

Since, for the moment, the Dusk block structure is still not set in stone, these functions have been left blank. As for transactions, they have been created with the mock StealthTx object included in the `transactions` folder. These will most likely also be subject to change once we formally decide on a transaction structure, and build out that library more.