# Dusk Network Testnet [![Build Status](https://travis-ci.org/dusk-network/dusk-blockchain.svg?branch=testnet)](https://travis-ci.com/dusk-network/dusk-blockchain)
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
This guide is for building the node from source. If you would like to just download the compiled program, head over to the [releases](https://github.com/dusk-network/dusk-blockchain/releases) page, which should include both a pre-built DUSK node, and a pre-built blind bid executable.

NOTE: This guide assumes you are building and running from a UNIX-like operating system. The node is not tested on Windows.

### Requirements
[Go](https://golang.org/) 1.13 or newer.

Optional - if you wish to participate in consensus: [the latest version of Rust](https://www.rust-lang.org/tools/install)

### Installation 
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

OPTIONAL: If you wish to participate in consensus, it is necessary that you also build and run the `blindbid` executable. Instructions for building the `blindbid` module can be found [here](https://github.com/dusk-network/dusk-blindbidproof/blob/master/Readme.md#how-to-build). After building, make sure you run it before starting the DUSK node.

And finally, to start your node, type:
```bash
./start-node.sh
```

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
Note that you can always get an overview of all commands and a short description about each of them, by typing `help` into the console.
### How to create a wallet?
In the console, type `createwallet [password]`, where `[password]` stands for the secret combination of user-choice.
### How to create a wallet from a seed?
Type `createfromseed [seed] [password]`, where `[seed]` stands for a hex seed and `[password]` stands for the secret combination of user-choice.
### How to load a wallet?
Type `loadwallet [password]`, where `[password]` stands for the secret combination previously selected by the user.
### How to check the balance of the address?
Simply type `balance` into the console, and the node will show you your locked and unlocked balances.
### How to claim Testnet DUSK?
To claim Testnet DUSK (tDUSK), the user is required to make a Twitter post containing his/her wallet address ([example](https://twitter.com/ellie12496641/status/1147604746280361984)). Following the post on Twitter, the user should go to the faucet [webpage](https://faucet.dusk.network/) and paste the Twitter post link into the empty box and click the `Send Dusk!` button. The tDUSK will be deposited onto the aforementioned address within a minute. The user can claim tDUSK for the same address once per 24 hours.
### How to transfer tDUSK?
Type `transfer [amount] [address]`, where `[amount]` stands for the amount of tDUSK the user is willing to bid (`0 < amount <= balance`), and `[address]` stands for the recipient address. Note that a wallet needs to be loaded for this to work.
### How to participate in consensus?
Once the node is started and a wallet is loaded, the node will automatically begin staking and bidding tDUSK once it finds some in the wallet. Once these transactions are included in a valid block, the node will begin performing it's consensus-related duties. Note that the values and intervals are standardized inside of the node configuration file (`dusk.toml`), and can be adjusted to fit the needs of the user.
