# [harness](./harness)

Test harness for simulation network testing.

<!-- ToC start -->
##  Contents

   1. [Test-Harness](#test-harness)
      1. [Harness engine](#harness-engine)
         1. [Utilities to manipulate a running node](#utilities-to-manipulate-a-running-node)
         1. [Utilities to monitor a running node](#utilities-to-monitor-a-running-node)
      1. [Directory structure](#directory-structure)
         1. [HowTo](#howto)
            1. [Configure](#configure)
            1. [Run](#run)
<!-- ToC end -->

## Test-Harness

### Harness engine

Test harness is here to allow automating general-purposes and complex E2E
dusk-network testing.

A common structure of such test is:

1. Define configuration for each network node
2. Bootstrap a network of N nodes, all properly configured
3. Perform change-state actions \(e.g send transaction, send wire message etc
   ...\)
4. Start monitoring utilities
5. Perform checks to ensure proper state/result has been achieved

#### Utilities to manipulate a running node

* `engine.*Cmd` - a set of gRPC calls to node gRPC server
* `engine.PublishTopicCmd` - a gRPC call to inject a message into the eventBus
  \(pending\)
* `engine.SendWireMsg` - send a message to P2P layer

#### Utilities to monitor a running node

* `engine.SendQuery` - send graphql query to a specified node to fetch node data

### Directory structure

`Local network workspace` - a temporary folder that hosts all nodes directories
during the test execution

```bash
ls /tmp/localnet-429879163                                                                                 
node-9000  node-9001  node-9002  node-9003
```

`Node directory` - a temporary folder that hosts all data relevant to the
running node

```bash
$ ls /tmp/localnet-429879163/node-9001/
chain  dusk7001.log  dusk-grpc.sock dusk.toml  pipe-channel  walletDB
```

#### HowTo

##### Configure

Considering that you have previously cloned and built `dusk-blindbidproof`
, `dusk-seeder` in the root directory as `dusk-blockchain`, configure your env
vars like this:

```bash
DUSK_HOME=/opt/gocode/src/github.com/dusk-network
DUSK_BLINDBID=$DUSK_HOME/dusk-blindbidproof/target/debug/blindbid
DUSK_BLOCKCHAIN=$DUSK_HOME/dusk-blockchain/bin/dusk
DUSK_SEEDER=$DUSK_HOME/dusk-seeder/voucher
DUSK_WALLET_PASS="default"
```

##### Run

```bash
tests$ go test -v --count=1 --test.timeout=0  ./... -args -enable
```

Alternatively, you can have a one liner do it all \(run it from dusk-blockchain
root
dir\): `DUSK_BLOCKCHAIN=$PWD/bin/dusk DUSK_BLINDBID=$PWD/../dusk-blindbidproof/target/debug/blindbid DUSK_SEEDER=$PWD/../dusk-seeder/voucher DUSK_WALLET_PASS="default" make test-harness`

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
