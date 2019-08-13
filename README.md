# Dusk Network Testnet 
## Shin (殉星) Release
|Table of Contents|
|---|
|[What is the Dusk Network Testnet Shin (殉星)?](#what-is-the-dusk-network-testnet-shin)|
|[Specification Requirements](#specification-requirements)|
|[Installation Guide](#installation-guide)|
|[Features](#features)|
|[Upcoming Features](#upcoming-features)|
|[How to use the wallet?](#how-to-use-the-wallet)|
## What is the Dusk Network Testnet Shin (殉星) ?
Dusk Network Testnet Shin (殉星) is the first publicly-available implementation of the Dusk Network protocol. Dusk Network is a privacy-oriented blockchain protocol, that anyone can use to create zero-knowledge dApps. The Dusk Network protocol is secured via Segregated Byzantine Agreement consensus protocol. Segregated Byzantine Agreement is a permission-less consensus protocol with statistical block finality. Segregated Byzantine Agreement also includes Proof of Blind Bid, a novel Private Proof-of-Stake implementation, that enables Block Generators to stake anonymously.
## Specification Requirements
### Minimum Specifications
| CPU | RAM | Storage | Network Connection |
|---|---|---|---|
|2 cores; 2 GHz| 1 GB | 60 GB | 1 Mbps |
### Recommended Specifications
| CPU | RAM | Storage | Network Connection |
|---|---|---|---|
|4 cores; 2 GHz| 4 GB | 250 GB | 10 Mbps |
## Installation Guide
This guide is for building the node from source. If you would like to just download the compiled program, head over to the releases tab. 

NOTE: This guide assumes you are building and running from a UNIX-like operating system. The node is not tested on Windows.

### Requirements
[Go](https://golang.org/) 1.11 or newer.

Optional - if you wish to participate in consensus: [Rust](https://www.rust-lang.org/tools/install)

### Installation 
#### DUSK node
First, download the codebase and it's dependencies into your $GOPATH by running:
```bash
go get github.com/dusk-network/dusk-blockchain
```
Then, navigate to the testnet folder, like so:
```bash
cd $GOPATH/src/github.com/dusk-network/dusk-blockchain/launch/testnet
```
Then, to build the binary, simply run:
```bash
go build
```

And finally, to start your node, type:
```bash
./testnet
```

If you wish to participate in consensus, it is necessary that you also build and run the `blindbid` executable, explained below. If not, feel free to skip that section.
#### Blind bid 
Instructions for building the `blindbid` module can be found [here](https://github.com/dusk-network/dusk-blindbidproof/blob/master/Readme.md#how-to-build).
## Features
1. Cryptography Module - Includes an implementation of SHA-3 and LongsightL hash functions, Ristretto and BN-256 elliptic curves, Ed25519, BLS, bLSAG and MLSAG signature schemes, Bulletproofs zero-knowledge proof scheme.
2. Consensus Module - Includes a complete implementation of the latest version of Segregated Byzantine Agreement (v2.1) consensus protocol, which contains three phases - Block Generation, Block Reduction and Block Agreement as well as the Blind Bid proof protocol utilized in the Block Generation phase.
3. Networking and Database Module - Includes the blockchain storage and related logic, as well as the implementation of P2P gossip protocol utilized by the Dusk Network protocol.
4. CLI Wallet - Includes an functionality enabling the user to create/load a wallet, transfer/bid/stake DUSK tokens. The instructions on each of the aforementioned functionalities will be listed below. 
## Upcoming Features
These features will be introduced in the later iterations of the Testnet (starting from v2).
1. Virtual Machine Module - Will include the implementation of the WebAssembly-based (WASM) Turing-complete Virtual Machine with zero-knowledge proof verification capabilities, as well as an account-based state-layer confidential transaction model and the implementation of XSC standard.
2. Guru Module - Will include the reputation module incorporated to aid the consensus combat malicious behaviour.
3. Cryptoeconomics Model - Will include an optimized reward mechanism for the consensus alongside the slashing conditions as well as introducing the lower and upper thresholds for the bids and stakes. The cryptoeconomics model will also include the rolling transaction fee model.
4. Poseidon Hash Function - Will include the Poseidon hash function which will supersede LongsightL as the new zero-knowledge proof-friendly hash function.
5. Anonymous Networking Layer - Will include the anonymous P2P communication model based on onion routing.
## How to use the wallet?
A video tutorial can be found here: https://youtu.be/VWP-IY31jxI
### How to create a wallet?
Open the Command Line Interface (CLI) and type `createwallet [password]`, where `[password]` stands for the secret combination of user-choice.
### How to create a wallet from a seed?
Open the Command Line Interface (CLI) and type `createfromseed [seed] [password]`, where `[seed]` stands for a hex seed and `[password]` stands for the secret combination of user-choice.
### How to load a wallet?
Open the Command Line Interface (CLI) and type `loadwallet [password]`, where `[password]` stands for the secret combination previously selected by the user.
### How to check the balance of the address?
Open the Command Line Interface (CLI) and type `balance`. 
### How to claim Testnet DUSK?
To claim Testnet DUSK (tDUSK), the user is required to make a Twitter post containing his/her wallet address ([example](https://twitter.com/ellie12496641/status/1147604746280361984)). Following the post on Twitter, the user should go to the faucet [webpage](https://faucet.dusk.network/) and paste the Twitter post link into the empty box and click the `Send Dusk!` button. The tDUSK will be deposited onto the aforementioned address within a minute. The user can claim tDUSK for the same address once per 24 hours.
### How to transfer tDUSK?
Open the Command Line Interface (CLI) and type `transfer [amount] [address] [password]`, where `[amount]` stands for the amount of tDUSK the user is willing to bid (`0 < amount <= balance`), `[address]` stands for the recipient address, and `[password]` stands for the wallet password. 
### How to become a Provisioner?
Open the Command Line Interface (CLI) and type `stake [amount] [locktime] [password]`, where `[amount]` stands for the amount of tDUSK the user is willing to stake (`0 < amount <= balance`), `[locktime]` stands for the amount of blocks for which the stake is locked (`0 < locktime < 250000`), and `[password]` stands for the wallet password. 

After completion, type `startprovisioner` into the CLI to join the consensus.
### How to become a Block Generator?
Open a Command Line Interface (CLI) and type `bid [amount] [locktime] [password]`, where `[amount]` stands for the amount of tDUSK the user is willing to bid (`0 < amount <= balance`), `[locktime]` stands for the amount of blocks for which the bid is locked (`0 < locktime < 250000`), and `[password]` stands for the wallet password. 

After completion, you should note down the tx hash that the wallet prints out. Wait a minute or two for the transaction to be included in a block, and then type `startblockgenerator [txid]` into the CLI to join the consensus.
