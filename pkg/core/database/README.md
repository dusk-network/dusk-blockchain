### General concept
Package database represents a database layer in Dusk Network. Current design should embody both backend-agnostic and storage-agnostic concepts.

###  Terminology

`Storage`: Underlying storage can be any kind of KV stores (BoltDB, LMDB, LevelDB, RocksDB, Redis), non-KV stores (SQLite) or even a custom ACID-complient flat-file store.

`Backend/Driver`: represents a DB implementation that uses one or multiple underlying storages to provide block chain database specifics with regard to particular purpose. It must implement Driver, DB and Tx intefaces.

Interfaces exposed to upper layers:

- `database.Driver`: DB connection initiator. Inspired by database/sql/driver
- `database.DB`: A thin layer on top of database.Tx providing a manageable Tx execution. Inspired by BoltDB
- `database.Tx`: A transaction layer tightly coupled with DUSK block chain specifics to fully benefit from underlying storage capabilities


### Available Drivers

- `/database/heavy` driver is designed to provide efficient, robust and persistent DUSK block chain DB on top of syndtr/goleveldb/leveldb store (unofficial LevelDB porting). It must be Mainnet-complient.

- `/database/lite` driver is designed to provide human-readable DUSK block chain DB for DevNet needs only. As is based on SQLite3, any SQL browser can be used as blockchain explorer.

### Code example:

```

// By changing Driver type, one can switch between different backends
// Each backend is bound to one or multiple underlying stores
databaseDriverType := "Lite"
readonly := false

// Retrieve
driver := database.From(databaseDriverType)
db, err := driver.Open(path, protocol.DevNet, readonly)

h := &block.Header{}
// Populate h

// a managed read-write DB Tx
err = db.Update(func(tx database.Tx) error {
	err := tx.WriteHeader(h)
	return err
})

// a managed read-only DB Tx
retrievedHeight := 0
err = db.View(func(tx database.Tx) error {
	header, err := tx.GetBlockHeaderByHash(h.Hash)
	retrievedHeight = header.Height
	return err
})

db.Close()

```

### Additional features

Additional features that can be provided by a Driver:

- Read-only transactions/connections
- Validated storage connection
- Profiling: Collect transactions and connections statistics
- Traces: Log transactions data
- Alarming: Trigger an event in case of critical backend or storage failure
- Iterators/Cursors
- Thread-safety: One read-write transaction, many read-only transactions


### Pending development

- Implement a driver that uses RocksDB, LDBM or BoltDB as a KV store. This driver should serve as a second option for Mainnet needs. 