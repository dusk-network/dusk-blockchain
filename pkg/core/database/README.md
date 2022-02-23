# README

## General concept

Package database represents a database layer in Dusk Network. Current design should embody both backend-agnostic and storage-agnostic concepts.

## Terminology

`Storage`: Underlying storage can be any kind of KV stores \(BoltDB, LMDB, LevelDB, RocksDB, Redis\), non-KV stores \(SQLite\) or even a custom ACID-complient flat-file store.

`Backend/Driver`: represents a DB implementation that uses one or multiple underlying storages to provide block chain database specifics with regard to particular purpose. It must implement Driver, DB and Tx intefaces.

Interfaces exposed to upper layers:

* `database.Driver`: DB connection initiator. Inspired by database/sql/driver
* `database.DB`: A thin layer on top of database.Tx providing a manageable Tx execution. Inspired by BoltDB
* `database.Tx`: A transaction layer tightly coupled with DUSK block chain specifics to fully benefit from underlying storage capabilities

## Available Drivers

* `/database/heavy` driver is designed to provide efficient, robust and persistent DUSK block chain DB on top of syndtr/goleveldb/leveldb store \(unofficial LevelDB porting\). It must be Mainnet-complient.

## Testing Drivers

* `/database/testing` implements a boilerplate method to verify if a registered driver does satisfy minimum database requirements. The package defines a set of unit tests that are executed only on registered drivers. It can serve also as a detailed and working database guideline.

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

## Blockchain registry
Blockchain database supports (read/write) two additional KV records which store both hash of the blockchain latest block and hash of blockchain latest `persisted` block. A block has a meaning of a persisted block only if rusk.Persist grpc has been called for it successfully.

`FetchRegistry` method from Transaction interface should be used to fetch the abovementioned KV records.

## Additional features

Additional features that can be provided by a Driver:

* Read-only transactions/connections
* Validated storage connection
* Profiling: Collect transactions and connections statistics
* Traces: Log transactions data
* Alarming: Trigger an event in case of critical backend or storage failure
* Iterators/Cursors
* Safe Concurrency model: One read-write transaction, many read-only transactions
* Redundancy: blockchain data stored in a secondary in-memory or on-disk structure

