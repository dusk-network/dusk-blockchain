# Description
Below are a few snippets how to fetch log details per a specified context. The snippets are based on `logger.format=json`.

# Logging grep snippets

## General (cross-subsystem)
```bash
# Monitor for block accepting events
tail -F /var/log/dusk.log | grep -e 'agreement_received\|accept_block' 

# Fetch all errors and warnings
grep -e "error\|warn" /var/log/dusk.log

# Fetch all traces per a block hash (candidate or accepted)
grep  '"hash":"bdc02411' /var/log/dusk.log

# Fetch all traces per a block height
grep  '"height":"99999' /var/log/dusk.log

```
## P2P subsystem
```bash
# Fetch all traces from P2P network connections.  
grep ':"peer"'  /var/log/dusk.log

# Fetch all errors and warnings from established and disconnected peers.  
grep ':"peer"'  /var/log/dusk.log | grep -e "error\|warn"

# Fetch all messages and errors originated from peer 178.62.216.99:7100
grep ':"peer"'  /var/log/dusk.log | grep 'r_addr":"178.62.216.99:7100'

# Fetch all inbound and outbound peer connections state (established/terminated)
grep "peer_connection" /var/log/dusk.log
```

## Consensus subsystem
```bash

#  Fetch all traces from Consensus
grep ':"consensus"'  /var/log/dusk.log

# Fetch all errors and warnings reported by consensus.  
grep ':"consensus"'  /var/log/dusk.log | grep -e "error\|warn"

# Fetch all consensus errors and warnings for round 10
grep 'round":10'   /var/log/dusk.log | grep -e "error\|warn"

# Fetch consensus state at agreement_received event for round 10
grep 'round":10'   /var/log/dusk.log | grep 'agreement_received'

```


## Synchronizer subsystem
```bash
#  Fetch all traces from Synchronizer
grep ':"sync"'  /var/log/dusk.log

# Fetch all errors and warnings from Synchronizer  
grep ':"sync"'  /var/log/dusk.log | grep -e "error\|warn"

# Fetch all Synchronizer state transitions
grep ':"sync"'  /var/log/dusk.log | grep "change state"
```
## Chain/Database subsystem
```bash
#  Fetch all traces from Chain
grep ':"chain"'  /var/log/dusk.log
```

## Mempool subsystem
```bash
#  Fetch all traces from Mempool
grep ':"mempool"'  /var/log/dusk.log
```

## GraphQL server subsystem
```bash
#  Fetch all traces from GraphQL server
grep ':"gql"'  /var/log/dusk.log
```

## gRPC server subsystem
```bash
#  Fetch all traces from gRPC server
grep ':"grpc_s"'  /var/log/dusk.log
```

## gRPC client to Rusk service
```bash
#  Fetch all traces from gRPC client calls
grep ':"grpc_c"'  /var/log/dusk.log
```

## Transactor subsystem
```bash
#  Fetch all traces from Transactor
grep ':"transactor"'  /var/log/dusk.log
```









## Kadcast subsystem
```bash
# TBD
```

## Monitoring
```bash
# Fetch number of goroutines
grep ':"memstats"'  /var/log/dusk.log | grep -e "NumGoroutine"
```

 
