# Dusk Network Node

Official Golang reference implementation of the DUSK Network protocol.

[![Actions Status](https://github.com/dusk-network/dusk-blockchain/workflows/Continuous%20Integration/badge.svg)](https://github.com/dusk-network/dusk-blockchain/actions) 
[![codecov](https://codecov.io/gh/dusk-network/dusk-blockchain/branch/master/graph/badge.svg)](https://codecov.io/gh/dusk-network/dusk-blockchain)
[![Go Report Card](https://goreportcard.com/badge/github.com/dusk-network/dusk-blockchain?style=flat-square)](https://goreportcard.com/report/github.com/dusk-network/dusk-blockchain)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/dusk-network/dusk-blockchain)](https://pkg.go.dev/github.com/dusk-network/dusk-blockchain)

## Specification Requirements

The following requirements are defined for running an active Dusk node. Depending on the role your node plays and how much functionality it exposes, the utilization of the node might vary significantly. 

### Minimum Specifications

| CPU | RAM | Storage | Network Connection |
| :--- | :--- | :--- | :--- |
| 4 cores; 2 GHz | 4 GB | 100 GB | 10 Mbps |

### Recommended Specifications

| CPU | RAM | Storage | Network Connection |
| :--- | :--- | :--- | :--- |
| 8 cores; 2 GHz | 8 GB | 250 GB | +25 Mbps |

## Installation Guide

This guide is for building the node from source. If you would like to just download the compiled program, head over to the [releases](https://github.com/dusk-network/dusk-blockchain/releases) page, which should include a pre-built DUSK node, and a pre-built wallet executable.

NOTE: This guide assumes you are building and running from a UNIX-like operating system. The node is not tested on Windows.

### Requirements

[Go](https://golang.org/) 1.17 or newer.

### Installation

Download the codebase and navigate into the folder:

```bash
git clone git@github.com:dusk-network/dusk-blockchain.git && cd dusk-blockchain
```

Get the project dependencies by running:

```bash
go get github.com/dusk-network/dusk-blockchain/...
```

To build the binary, simply run:

```bash
make build
```

Finally, to start your node, type:

```bash
./bin/dusk --config=dusk.toml
```

## Wallet

The wallet is hosted in a separate repository, [found here](https://github.com/dusk-network/wallet-cli). 

### How to use the wallet

For more information on how to install, configure and run the CLI wallet, see the documentation [here](https://github.com/dusk-network/wallet-cli/tree/main/src/bin).

## Rusk

Rusk is an important separate service that should be ran next to the node. Rusk is a powerful wrapper around the VM/execution engine that provides the genesis contracts and gives the VM access to host functions. Rusk is hosted in a separate repository, [found here](https://github.com/dusk-network/rusk).

### How to use Rusk

For more information on how to install, configure and run the Rusk, see the documentation [here](https://github.com/dusk-network/rusk#readme).

## License

The Dusk Network blockchain client is licensed under the MIT License. See [the license file](LICENSE) for details.

## Contributing

Please see [the contribution guidelines](CONTRIBUTING.md) for details.
