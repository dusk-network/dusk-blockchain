### General concept
Package database represents a database layer in Dusk Network. Current design should embody both backend-agnostic and storage-agnostic concepts.

###  Terminology

`Storage`: Underlying storage might be any of kind the KV stores like BoltDB, LMDB, LevelDB, RocksDB, Redis or non-KV stores SQLite or even a custom flat-file store.

`Backend/Driver`: represents a DB implementation that uses one or multiple underlying storages to provide Block chain database specifics with regard to particular purpose. It must satisfy Driver, DB and Tx intefaces.

Interfaces exposed to upper layers:

- `database.Driver`: DB connection initiator. Inspired by database/sql/driver
- `database.DB`: A thin layer on top of database.Tx providing a manageable Tx execution. Inspired by BoltDB
- `database.Tx`: A transaction layer tightly coupled with Blockchain specifics to fully benefit from underlying storage capabilities


### Drivers

- /database/heavy driver is designed to provide efficient, robust and persistent Dusk Blockchain DB on top of syndtr/goleveldb/leveldb store. It must be Mainnet-complient.

- /database/lite driver is designed to provide human-readable Dusk Blockchain DB for Devnet needs only. As is based on sqlite, any SQL browser can be used as blockchain explorer.

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

Supported features:

- Read-only transactions/connections
- Validated storage conn
- (pending) Profiling / Stats to collect statistics around latency and history of transactions
- (pending) Traces to log transactions data for DevNet/TestNet needs
- (pending) Alarming to trigger an event to upper layers in case of critical Backend or Storage failure
- (pending) Iterators/Cursors
- TODO