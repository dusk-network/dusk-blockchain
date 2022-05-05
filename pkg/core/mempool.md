# README

## Mempool

Mempool represents a pool of valid transactions pushed by the network, ready to be included in the next candidate block. 


### Main responsibilities:

* Queue transactions received from the network, ready to be verified
* Verify any received transaction
* Store transactions in the  mempool state that do pass fully acceptance criteria
* On block acceptance, remove all accepted transactions from the `mempool state`

### Transaction acceptance criteria:

- Decodable 
- Passes `rusk.Preverify`
- Does not exist in the `mempool state`
- Does not exist in the `blockchain state`
- Does not contain a nullifier already used by another transaction in mempool state.

### Underlying storage

Mempool is storage-agnostic. An underlying storage must implement interface `Pool` to be applicable. At that stage, we support two types of stores:

* hashmap - based on golang map that implements in-memory key/value store.
* buntdb - based on buntdb, a low-level, in-memory and ACID compliant key/value store that persists to disk.

## Implementation details

### Exposed methods
Mempool exposes `ProcessTx` method that is concurrent-safe, implements trasaction acceptance criteria and adds valid transaction to the mempool state.

### Background goroutines
Mempool is driven by two goroutines:

`Main Loop` goroutine handles:
- GetMempoolTransactions request from a Block Generator component
- Block Accepted event

`PropagateLoop` goroutine handles:
- any request for transaction repropagation sent by `ProcessTx`.





