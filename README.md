# Dusk Network Testnet [![Build Status](https://travis-ci.org/dusk-network/dusk-blockchain.svg?branch=testnet)](https://travis-ci.com/dusk-network/dusk-blockchain)

## Rei (レイ) Release

|Table of Contents|
|---|
|[What is Dusk Network Testnet Rei (レイ)?](#what-is-dusk-network-testnet-rei)|
|[Specification Requirements](#specification-requirements)|
|[Installation Guide](#installation-guide)|
|[Features](#features)|
|[Upcoming Features](#upcoming-features)|
|[How to use the wallet?](#how-to-use-the-wallet)|

## What is Dusk Network Testnet Rei (レイ)?

Dusk Network Testnet Rei (レイ) is the second iteration of the publicly-available implementation of the Dusk Network protocol. This iteration addresses stability issues and refines the user experience for node operators. It features an all-new upgraded design for the consensus implementation, as well as a more user-friendly way of managing the node and using the wallet.

Dusk Network is a privacy-oriented blockchain protocol, that anyone can use to create zero-knowledge dApps. The Dusk Network protocol is secured via Segregated Byzantine Agreement consensus protocol. Segregated Byzantine Agreement is a permission-less consensus protocol with statistical block finality. Segregated Byzantine Agreement also includes Proof of Blind Bid, a novel Private Proof-of-Stake implementation, that enables Block Generators to stake anonymously.

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

This guide is for building the node from source. If you would like to just download the compiled program, head over to the [releases](https://github.com/dusk-network/dusk-blockchain/releases) page, which should include a pre-built DUSK node, a pre-built blind bid executable, and a pre-built wallet executable.

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
cd $GOPATH/src/github.com/dusk-network/dusk-blockchain
```
Then, to build the binary, simply run:
```bash
make build
```

Next up, we build the rust part of the node. Instructions for building the `blindbid` module can be found [here](https://github.com/dusk-network/dusk-blindbidproof/blob/master/Readme.md#how-to-build). After building, make sure you move the executable into the `testnet` folder.

Finally, to start your node, type:
```bash
./bin/dusk --config=dusk.toml
```

## Features

1. Cryptography Module - Includes an implementation of SHA-3 and LongsightL hash functions, Ristretto and BN-256 elliptic curves, Ed25519, BLS, bLSAG and MLSAG signature schemes, Bulletproofs zero-knowledge proof scheme.
2. Consensus Module - Includes a complete implementation of the latest version of Segregated Byzantine Agreement (v2.1) consensus protocol, which contains three phases - Block Generation, Block Reduction and Block Agreement as well as the Blind Bid proof protocol utilized in the Block Generation phase.
3. Networking and Database Module - Includes the blockchain storage and related logic, as well as the implementation of P2P gossip protocol utilized by the Dusk Network protocol.
4. CLI Wallet - Includes an functionality enabling the user to create/load a wallet, transfer/bid/stake DUSK tokens, and managing the node. 

## Upcoming Features

These features will be introduced in the later iterations of the Testnet (starting from v2).
1. Virtual Machine Module - Will include the implementation of the WebAssembly-based (WASM) Turing-complete Virtual Machine with zero-knowledge proof verification capabilities, as well as an account-based state-layer confidential transaction model and the implementation of XSC standard.
2. Guru Module - Will include the reputation module incorporated to aid the consensus combat malicious behavior.
3. Cryptoeconomics Model - Will include an optimized reward mechanism for the consensus alongside the slashing conditions as well as introducing the lower and upper thresholds for the bids and stakes. The cryptoeconomics model will also include the rolling transaction fee model.
4. Poseidon Hash Function - Will include the Poseidon hash function which will supersede LongsightL as the new zero-knowledge proof-friendly hash function.
5. Anonymous Networking Layer - Will include the anonymous P2P communication model based on onion routing.

## How to use the wallet?

The wallet is hosted in a separate repo, [found here](https://github.com/dusk-network/dusk-wallet-cli). Please refer to that repository for build instructions.

After building, ensure you move the `dusk-wallet-cli` executable into the same folder as your `testnet`, `blindbid` and `dusk.toml` files. You can start the wallet after launching the node.

```bash
./dusk-wallet-cli
```

The wallet will show you a menu with available options, that you can navigate with the arrow keys and the enter key.

Note that the wallet is a seperate process from the node, and thus closing the wallet does not stop the node from running.

### How to claim Testnet DUSK?

To claim Testnet DUSK (tDUSK), the user is required to make a Twitter post containing his/her wallet address ([example](https://twitter.com/ellie12496641/status/1147604746280361984)). Following the post on Twitter, the user should go to the faucet [webpage](https://faucet.dusk.network/) and paste the Twitter post link into the empty box and click the `Send Dusk!` button. The tDUSK will be deposited onto the aforementioned address within a minute. The user can claim tDUSK for the same address once per 24 hours.
