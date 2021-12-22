
# Dusk Network Node

Official reference implementation of the DUSK Network protocol in Golang.

[![Actions Status](https://github.com/dusk-network/dusk-blockchain/workflows/Continuous%20Integration/badge.svg)](https://github.com/dusk-network/dusk-blockchain/actions)
[![codecov](https://codecov.io/gh/dusk-network/dusk-blockchain/branch/master/graph/badge.svg)](https://codecov.io/gh/dusk-network/dusk-blockchain)
[![Go Report Card](https://goreportcard.com/badge/github.com/dusk-network/dusk-blockchain?style=flat-square)](https://goreportcard.com/report/github.com/dusk-network/dusk-blockchain)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/dusk-network/dusk-blockchain)](https://pkg.go.dev/github.com/dusk-network/dusk-blockchain)
<!-- ToC start -->
## Contents

1. [Specification Requirements](#specification-requirements)
    1. [Minimum Specifications](#minimum-specifications)
    1. [Recommended Specifications](#recommended-specifications)
1. [Installation Guide](#installation-guide)
    1. [Requirements](#requirements)
    1. [Installation](#installation)
1. [Wallet](#wallet)
    1. [Building the wallet](#building-the-wallet)
    1. [Running the wallet](#running-the-wallet)
    1. [How to use the wallet](#how-to-use-the-wallet)
1. [License](#license)
1. [Contributing](#contributing)
<!-- ToC end -->
## Specification Requirements

### Minimum Specifications

| CPU | RAM | Storage | Network Connection |
| :--- | :--- | :--- | :--- |
| 2 cores; 2 GHz | 1 GB | 60 GB | 1 Mbps |

### Recommended Specifications

| CPU | RAM | Storage | Network Connection |
| :--- | :--- | :--- | :--- |
| 4 cores; 2 GHz | 4 GB | 250 GB | 10 Mbps |

## Installation Guide

This guide is for building the node from source. If you would like to just download the compiled program, head over to the [releases](https://github.com/dusk-network/dusk-blockchain/releases) page, which should include a pre-built DUSK node, and a pre-built wallet executable.

NOTE: This guide assumes you are building and running from a UNIX-like operating system. The node is not tested on Windows.

### Requirements

[Go](https://golang.org/) 1.17 or newer.

### Installation

First, download the codebase and it's dependencies into your $GOPATH by running:

```bash
go get github.com/dusk-network/dusk-blockchain
```

Then, navigate to the testnet folder, like so:

```bash
cd $GOPATH/src/github.com/dusk-network/dusk-blockchain
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

The wallet is hosted in a separate folder, [found here](./cmd/wallet).

### Building the wallet

The wallet is automatically built when running `make build`. You can then execute it by typing:

```bash
./bin/wallet
```


### Running the wallet

Alternatively, to build and run the wallet in a single command, simply type:

```bash
make wallet
```

### How to use the wallet

The wallet will show you a menu with available options, that you can navigate with the arrow keys and the enter key.

Note that the wallet is a seperate process from the node, and thus closing the wallet does not stop the node from running.

## License

The Dusk Network blockchain client is licensed under the MIT License. See [the license file](LICENSE) for details.

## Contributing

Please see [the contribution guidelines](CONTRIBUTING.md) for details.
