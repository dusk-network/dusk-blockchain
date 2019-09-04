
#### Test harness engine
 
Test harness is here to allow automating general-purposes and complex E2E dusk-network testing. 

A common structure of such test is:

0. Define configuration for each network node
1. Bootstrap a network of N nodes, all properly configured
2. Perform change-state actions (e.g send transaction, send wire message etc ...)
3. Start monitoring utilities
4. Perform checks to ensure proper state/result has been achieved


###### Utilities to manipulate a running node

- `engine.PublishTopic` - injects a message into the eventBus of a specified node
- `engine.SendCommand` - sends a rpc command to a specified node for state-changing/data-mutation
- `engine.SendWireMsg` - send a message to P2P layer of a specified node (pending)

###### Utilities to monitor a running node 
- `engine.SendQuery` - send graphql query to a specified node to fetch node data
- `engine.GetMetrics` - send graphql query to a specified node to fetch node Performance metrics/stats (pending)
 

#### Directory structure

`Local network workspace` - a temporary folder that hosts all nodes directories during the test execution
```bash
ls /tmp/localnet-429879163                                                                                 
node-9000  node-9001  node-9002  node-9003
```

`Node directory` - a temporary folder that hosts all data relevant to the running node
```bash
$ ls /tmp/localnet-429879163/node-9001/
chain  dusk7001.log  dusk.toml  pipe-channel  walletDB
``` 



##### HowTo

###### Configure

The following ENV variables must be declared to enable testing.

```bash
DUSK_BLINDBID="/dusk-blockchain/launch/testnet/blindbid-avx2"
DUSK_BLOCKCHAIN="/dusk-blockchain/launch/testnet/testnet"
DUSK_SEEDER="dusk-voucher-seeder/dusk-voucher-seeder"
DUSK_WALLET_PASS="default"
```

###### Run
```bash
tests$ go test -v --count=1 --test.timeout=0  ./...
```

 