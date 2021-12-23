# [pkg/core/database](./pkg/core/database)

An abstract driver interface for enabling multiple types of storage strategy
suited to different types of data, *and a blockchain data specific interface* -
with the purpose being data access workload optimised storage. TODO: this merges
two independent incomplete documents with the same intent, even a header is
duplicated.

<!-- ToC start -->

## Contents

1. [General concept](#general-concept)
1. [Terminology](#terminology)
1. [Available Drivers](#available-drivers)
1. [Testing Drivers](#testing-drivers)
1. [Code example:](#code-example:)
1. [Additional features](#additional-features)
1. [Drivers](#drivers)
    1. [Light Driver](#light-driver)
    1. [Heavy Driver](#heavy-driver)
1. [General concept](#general-concept-1)
1. [K/V storage schema to store a single `pkg/core/block.Block` into blockchain](#k/v-storage-schema-to-store-a-single-pkg/core/blockblock-into-blockchain)
1. [K/V storage schema to store a candidate `pkg/core/block.Block`](#k/v-storage-schema-to-store-a-candidate-pkg/core/blockblock)
1. [K/V storage schema to store block generator bid values](#k/v-storage-schema-to-store-block-generator-bid-values)

<!-- ToC end -->

## General concept

Package database represents a database layer in Dusk Network. Current design
should embody both backend-agnostic and storage-agnostic concepts.

## Terminology

`Storage`: Underlying storage can be any kind of KV stores \(BoltDB, LMDB,
LevelDB, RocksDB, Redis\), non-KV stores \(SQLite\) or even a custom
ACID-complient flat-file store.

`Backend/Driver`: represents a DB implementation that uses one or multiple
underlying storages to provide block chain database specifics with regard to
particular purpose. It must implement Driver, DB and Tx intefaces.

Interfaces exposed to upper layers:

* `database.Driver`: DB connection initiator. Inspired by database/sql/driver
* `database.DB`: A thin layer on top of database.Tx providing a manageable Tx
  execution. Inspired by BoltDB
* `database.Tx`: A transaction layer tightly coupled with DUSK block chain
  specifics to fully benefit from underlying storage capabilities

## Available Drivers

* `/database/heavy` driver is designed to provide efficient, robust and
  persistent DUSK block chain DB on top of syndtr/goleveldb/leveldb store
  \(unofficial LevelDB porting\). It must be Mainnet-complient.

## Testing Drivers

* `/database/testing` implements a boilerplate method to verify if a registered
  driver does satisfy minimum database requirements. The package defines a set
  of unit tests that are executed only on registered drivers. It can serve also
  as a detailed and working database guideline.

## Code example:

More code examples can be found in `/database/heavy/database_test.go`

```text
// By changing Driver name, one can switch between different backends
// Each backend is bound to one or multiple underlying stores
readonly := false

// Retrieve
driver, _ := database.From(lite.DriverName)
db, err := driver.Open(path, protocol.DevNet, readonly)

if err != nil {
    ...
}

defer db.Close()

blocks := make([]*block.Block, 100)
// Populate blocks here ...

// a managed read-write DB Tx to store all blocks via atomic update
err = db.Update(func(tx database.Tx) error {
    for _, block := range blocks {
        err := tx.StoreBlock(block)
        if err != nil {
            return err
        }
    }
    return nil
})

if err != nil {
    fmt.Printf("Transaction failed. No blocks stored")
    return
}

 // a managed read-only DB Tx to check if all blocks have been stored
_ = db.View(func(tx database.Tx) error {
    for _, block := range blocks {
        exists, err := tx.FetchBlockExists(block.Header.Hash)
        if err != nil {
            fmt.Printf(err.Error())
            return nil
        }

        if !exists {
            fmt.Printf("Block with Height %d was not found", block.Header.Height)
            return nil
        }
    }
    return nil
})
```

## Additional features

Additional features that can be provided by a Driver:

* Read-only transactions/connections
* Validated storage connection
* Profiling: Collect transactions and connections statistics
* Traces: Log transactions data
* Alarming: Trigger an event in case of critical backend or storage failure
* Iterators/Cursors
* Safe Concurrency model: One read-write transaction, many read-only
  transactions
* Redundancy: blockchain data stored in a secondary in-memory or on-disk
  structure

## Drivers

Drivers are implementations which satisfy the database interface and provide
some kind of access method to read and write blockchain data.

### Light Driver

Lite package represents a database driver that provides in-memory blockchain
storage. That said, it does not provide persistence storage. For now, it is
applicable only for testing purposes like:

* running a unit-test
* running a benchmark test
* running a test harness

It still implements transaction layer requirements like atomicity, consistency
and isolation but no durability. At some point later it can be used as baseline
to compare performance results.

### Heavy Driver

## General concept

For general concept explanation one can refer to /pkg/core/database/README.md.
This document must focus on decisions made with regard to goleveldb specifics

## K/V storage schema to store a single `pkg/core/block.Block` into blockchain

| Prefix | KEY | VALUE | Count | Used by |
| :---: | :---: | :---: | :---: | :---: |
| 0x01 | HeaderHash | Header.Encode\(\) | 1 per block |  |
| 0x02 | HeaderHash + TxID | TxIndex + Tx.Encode\(\) | block txs count |  |
| 0x04 | TxID | HeaderHash | block txs count | FetchBlockTxByHash |
| 0x05 | KeyImage | TxID | sum of block txs inputs | FetchKeyImageExists |
| 0x03 | Height | HeaderHash | 1 per block | FetchBlockHashByHeight |
| 0x07 | State | Chain tip hash | 1 per chain | FetchState |

## K/V storage schema to store a candidate `pkg/core/block.Block`

| Prefix | KEY | VALUE | Count | Used by |
| :---: | :---: | :---: | :---: | :---: |
| 0x06 | HeaderHash + Height | Block.Encode\(\) | Many per blockchain | Store/Fetch/Delete CandidateBlock |

Table notation

* HeaderHash - a calculated hash of block header
* TxID - a calculated hash of transaction
* \'+' operation - denotes concatenation of byte arrays
* Tx.Encode\(\) - Encoded binary form of all Tx fields without TxID

## K/V storage schema to store block generator bid values

| Prefix | KEY | VALUE | Count | Used by |
| :---: | :---: | :---: | :---: | :---: |
| 0x08 | ExpiryHeight | D + K | 1 per bidding transaction made by user | FetchBidValues |

Copyright © 2018-2022 Dusk Network
[MIT Licence](https://github.com/dusk-network/dusk-blockchain/blob/master/LICENSE)
