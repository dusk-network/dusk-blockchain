## DUSK JSON-RPC API Reference Document

This document aims to cover the available JSON-RPC endpoints in completion, as well as providing an example of the expected JSON structures when making method calls to any node.

### Request/Response structures

#### JSON Request

An overview of the bare request object:

```
{
	method: "method",
	params: ["param1", "param2"],
}
```

A full example of making a request, using cURL:

```bash
 curl --data-binary '{"jsonrpc":"2.0","method":"method","params":["param1", "param2"]}' -H 'content-type:application/json;' http://127.0.0.1:9000
```

#### JSON Response

An overview of the bare response object:

```
{
	result: "result",
	error: "error",
}
```

### Available methods

#### Basic functionality (light nodes and full nodes)

| Method | Params | Description | Pre-requisites |
| ------ | ------ | ----------- | -------------- |
| `transfer` | \<amount\>, \<address\> | Sends a standard transactions of \<amount\> DUSK to \<address\>. Returns a TXID on success. | wallet loaded |
| `address` | | Returns the address of the loaded wallet. | wallet loaded |
| `createwallet` | \<password\> | Creates a wallet file, encrypted with \<password\> | no wallet loaded |
| `loadwallet` | \<password\> | Loads a wallet file at the default directory. | no wallet loaded |
| `createfromseed` | \<seed\>, \<password\> | Creates a wallet file using a given \<seed\>, encrypted with \<password\> | no wallet loaded |
| `balance` | | Returns the unlocked and locked balance of the loaded wallet. | wallet loaded |
| `unconfirmedbalance` | | Returns the amount of DUSK that is in the mempool for the loaded wallet. | wallet loaded |
| `txhistory` | | Returns the transaction history for the loaded wallet. | wallet loaded |
| `walletstatus` | | Returns whether or not the wallet is loaded, as a "boolean" (0 or 1) | none |
| `syncprogress` | | Returns to what degree the node is synced up with the rest of its peers, as a percentage. | none |

#### Extended functionality (full nodes only)

| Method | Params | Description | Pre-requisites |
| ------ | ------ | ----------- | -------------- |
| `bid` | \<amount\>, \<locktime\> | Sends a bid transaction of \<amount\> DUSK to self. The transaction will be locked for \<locktime\> blocks after being accepted into a block. Returns a TXID on success. | wallet loaded |
| `stake` | \<amount\> \<locktime\> | Sends a stake transaction of \<amount\> DUSK to self. The transaction will be locked for \<locktime\> blocks after being accepted into a block. Returns a TXID on success. | wallet loaded |
| `automateconsensustxs` | | Tells the node to automatically renew stakes and bids, to save the user the trouble. Values and locktimes are inferred from configuration file. Returns a string indicating success or failure. | wallet loaded |
| `viewmempool` | (optional) \<txtype or txid\> | Returns an overview of the mempool. Optionally, a caller can supply either a txtype (1 byte), or a txid(32 bytes) in order to filter for specific items. | none |
