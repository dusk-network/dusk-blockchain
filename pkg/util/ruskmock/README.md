# [pkg/util/ruskmock](./pkg/util/ruskmock)

This package contains a stand-in RUSK server, which can be used for integration
testing and DevNet launches, and replaces the Rust version of RUSK seamlessly,
thanks to the abstraction offered by Protocol Buffers and GRPC.

<!-- ToC start -->
##  Contents

   1. [Usage](#usage)
   1. [How it works](#how-it-works)
   1. [Future work](#future-work)
<!-- ToC end -->

## Usage

The test-harness and DevNet should use the RUSK mock server automatically, as of
now. To launch it in a stand-alone fashion, use the following commands:

```bash
make build
./bin/utils mockrusk --rusknetwork tcp --ruskaddress 127.0.0.1:10000
```

The preferred network and address can be chosen, by simply changing the
arguments to `rusknetwork` and `ruskaddress`. The server can be interacted with
by establishing GRPC connections to the given address and port, after which you
can call methods on the server.

## How it works

The mock RUSK server should provide all the endpoints needed for the node to run
uninterrupted. This means that it handles wallet management, key generation,
transaction building, and block validation.

Some of its functions can either succeed or fail depending on the input given.
Since the mock RUSK server can not actually validate this input, it is left
instead to a configuration object to decide what passes and what doesn't.

```golang
// Config contains a list of settings that determine the behavior
// of the `Server`.
type Config struct {
PassScoreValidation           bool
PassTransactionValidation     bool
PassStateTransition           bool
PassStateTransitionValidation bool
}
```

In a normal scenario, all of these are set to `true`, allowing for seamless
consensus execution.

The RUSK mock server uses the legacy wallet libraries under the hood in order to
maintain some kind of proper functionality regarding the transfer of DUSK and
the staking and bidding. Since the incoming data is structured in the way
described in [rusk-schema](https://github.com/dusk-network/rusk-schema/), these
structures are always converted first into legacy structures, by the conversion
functions in the `legacy` package. Please consult this package for
accurate [schemas](../legacy/README.md) on how the data is stored in each
structure.

Once these structures are decoded, the RUSK mock server uses the imported legacy
libraries to perform the required operations, in the context of each method. The
RUSK mock server is capable of accurately tracking incoming and outgoing DUSK,
and maintains an up-to-date provisioner set, which can be requested at any time.
It also provides functionality for creating three types of transactions (
Transfer, Bid and Stake), and allows for score generation and verification, in
order to allow for the blind bid lottery to run.

The RUSK mock server attempts to read the node's config file on startup, in
order to determine the wallet and database that it wishes to use. The server
maintains a proper `walletDB` for the wallet that it is started with, and all
operations related to transfer of funds are performed on this database.

## Future work

As we move closer towards integration with RUSK, the node will need to change
its behaviour a little bit, since some currently mocked procedures will need to
be performed properly. For the mock RUSK server, this currently means that
the `FindBid` method needs to be implemented - it currently does not do
anything, and just returns nil pointers. Since it is not necessary at this point
in time, this should not be an issue.

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
