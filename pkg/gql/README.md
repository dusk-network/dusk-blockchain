##### Overview
GraphQL package is here to provide a read-only access to any persistent/non-persistent node data.
It should allow fetching:

- chain data (block header and transactions)
- mempool state information
- wallet/account information (pending)
- node status (pending)

##### Scenarios

Scenarios where it's supposed to be useful:

- Blockchain explorer fetching chain, mempool or consensus data
- Test Harness ensuring chain state after a set of actions executed
- User retrieving data in curl-request manner

Package should not be used for any data mutations or node commanding. Usually these

##### Transport

Currently, it's over HTTP but later WebSocket support could be added to enable data fetching in publish-subscribe manner.
(e.g graphql service capable of sending updates on newly accepted block)

##### Pending development
- Queries schema is pending to be modified/enriched accordingly.
- Subscriptions for particular chain/consensus updates (block accepted, reward received etc)

#### Configuration
```toml
[gql]
# enable graphql service
enabled=true
port=9001
```

##### Example queries
-  Fetch block by hash with full set of supported fields

```graphql
{
  blocks(hash: "GU3RPuimCsAXqCxBwOLAJJjXX0h1Q1EHLzkqCF1GliA=" ) {
    header {
       height
       hash
       timestamp
       version
       seed
       prevblockhash
       txroot
    }
    transactions{
      txid
      txtype
    }
  }
}
```
- Fetch block fields by height

```graphql
{
  blocks( height: 456 ) {
    header {
       height
       hash
       timestamp
    }
    transactions{
      txid
      txtype
    }
  }
}
```

- Fetch local chain tip and current state of mempool in a single request.
```graphql
{
  blocks(height: -1 ) {
    header {
       height
       hash
       timestamp
    }
  },
  mempool(txid: "") {
      txid
      txtype
  },
}
``` 

- Fetch block header fields for range of blocks (from 116346 to 116348 height)
```graphql
{
  blocks(range: [116346,116348] ) {
    header {
       height
       hash
       timestamp
       version
    }
    transactions {
      txid
      txtype
    }
  }
```
- Fetch data of a single (accepted) transaction

```graphql
{
   transactions(txid: "f+u3gwnz5T7OlL+1NGW5q3zyPwg/djxrzsWfA/ysJ04=") {
      txid
      txtype
      blockhash
  }
}
```

- Fetch data of a set of transactions by txIDs

```graphql
{
  transactions(txids: ["3JTSG9tFTwfuYedolRZdc3p1jF/tOGjVjBiadDbeZPc=","WuXkPSuf/D741vKSpl3C8bvyh8cdXCZON1vh7hcBHsw="]) {
      txid
      txtype
      blockhash
  }
}
```

- Fetch first and last block timestamps

```graphql
{
  tip: blocks(height: -1) {
    header {
       height
       timestamp 
    }
  },
  genesis: blocks(height: 0) {
    header {
       height
       timestamp 
    }
  }
}
```
