# README

## Intro

GraphQL package is here to provide a read-only access to any persistent/non-persistent node data. It is intended to play the same role to DUSK Blockchain data as SQL to RDBMS, providing query flexibility and multiple-fetches in single a query.

 It should allow fetching:

* chain data \(block header and transactions\)
* mempool state information
* node status \(pending\)

### API Endpoints

* `/graphql` - data fetching
* `/ws` - websocket notifications, if TLS is disabled
* `/wss` - secure websocket notifications, if TLS is enabled

### Scenarios

Scenarios where it's supposed to be useful:

* Blockchain explorer fetching chain, mempool or consensus data
* Test Harness ensuring chain state after a set of actions executed
* User retrieving data in curl-request manner

The utility should not be used for any data mutations or node commanding.

## Configuration

```text
# GraphQL API service
[gql]
# enable graphql service
enabled=true
address="127.0.0.1:9001"

# enable/disable both HTTPS and WSS
enableTLS = false
# cert file path
certFile = ""
# key file path
keyFile = ""

# maximum requests per second 
# uniqueness of a request is based on: 
# Remote IP, Request method and path
maxRequestLimit = 20
```

## Example queries  

NB: The examples from below represent only query structures. To send a query as a http request the following schema must be used:

```javascript
{  
   "query":"query ($num: Int!) { transactions(last: $num) { txid txtype }}",
   "variables":{  
      "num": 10
   }
}
```

* Fetch block by hash with full set of supported fields

```graphql
{
  blocks(hash: "194dd13ee8a60ac017a82c41c0e2c02498d75f48754351072f392a085d469620" ) {
    header {
       height
       hash
       timestamp
       version
       seed
       prevblockhash
       statehash
       txroot
       reward
       feespaid
    }
    transactions{
      txid
      txtype
    }
  }
}
```

* Fetch block fields by height

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

* Fetch local chain tip and all pending mempool txs in a single request.

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

* Fetch block header fields for range of blocks \(from 116346 to 116348 height\)

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
  }
  ```

* Fetch all data of a single \(accepted\) transaction

```graphql
{
   transactions(txid: "194dd13ee8a60ac017a82c41c0e2c02498d75f48754351072f392a085d469620") {
      txid
      txtype
      blockhash
      blocktimestamp
      gaslimit
      gasprice
      gasspent

      output {
        pubkey
      }

      input {
        keyimage
      }
  }
}
```

* Fetch data of a set of accepted transactions by txIDs

```graphql
{
  transactions(txids: ["dc94d21bdb454f07ee61e76895165d737a758c5fed3868d58c189a7436de64f7","5ae5e43d2b9ffc3ef8d6f292a65dc2f1bbf287c71d5c264e375be1ee17011ecc"]) {
      txid
      txtype
      blockhash
      gaslimit
      gasprice
      gasspent
  }
}
```

* Fetch first and last block timestamps

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

* Fetch last 10 blocks accepted

```graphql
{
  blocks(last: 10) {
    header {
       height
       timestamp 
    }
    transactions{
      txid
    }
  }
}
```

* Fetch a slice of blocks by their hashes

  ```graphql
  {
    blocks(hashes: ["194dd13ee8a60ac017a82c41c0e2c02498d75f48754351072f392a085d469620","ba87ceec9f31ccfccabc9d2be9f753666e0ed559744f59d3f4bd2afd320f7b03"]) {
    header {
       height
       hash
    }
    transactions{
      txid
      txtype
    }
  }
  ```

* Fetch last/latest accepted 100 transactions \(type and size fields\)

  ```graphql
  { 
    transactions(last: 100) 
    { 
      txid
      txtype
      size
    }
  }
  ```

* Calculate count of blocks \(tip - old height\) since 1970-01-01T00:00:20+00:00

  ```graphql
  {
    tip: blocks(height: -1) {
        header {
            height
        }
    }
    old: blocks(since: "1970-01-01T00:00:20+00:00") {
        header {
            height
        }
    }
  }
  ```

- Calculate count of blocks (tip - old height) since 1970-01-01T00:00:20+00:00
```graphql
{
	tip: blocks(height: -1) {
		header {
			height
		}
	}
	old: blocks(since: "1970-01-01T00:00:20+00:00") {
		header {
			height
		}
	}
}
```
