# README

## Mempool

Mempool represents a pool of valid transactions pushed by the network, ready to be included in the next candidate block. It also prioritizes transactions by fee.


### Main responsibilities:

* Queue transactions received from the network, ready to be verified
* Verify any received transaction
* Store transactions in the  mempool state that do pass fully acceptance criteria
* Sort transactions in the mempool state by fee.
* On block acceptance, remove all accepted transactions from the `mempool state`
* On request from block generator, provide a set of transactions with highest fee, up to a specified total size.

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
Mempool exposes `ProcessTx` method that is concurrent-safe, implements transaction acceptance criteria and adds valid transactions to the mempool state.

### Background goroutines
Mempool is driven by two goroutines:

`Main Loop` goroutine handles:
- GetMempoolTxsBySize request from a Block Generator component.
- GetMempoolTxs request from GraphQL component.
- Block Accepted event triggered by Chain component.

`PropagateLoop` goroutine handles:
- any request for transaction repropagation sent by `ProcessTx` and repropagates at proper rate. It also should prioritize propagation by transaction fee (not implemented, see also #1240).





