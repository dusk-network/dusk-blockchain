## DUSK JSON-RPC API Reference Document

This document aims to cover the available JSON-RPC endpoints in completion, as well as providing an example of the expected JSON structures when making method calls to any node.

### Request/Response structures

#### JSON Request

An overview of the bare request object:

```JSON
{
	method: "method",
	params: ["param1", "param2"],
}
```

A full example, using cURL:

```bash

```

#### JSON Response

An overview of the bare response object:

```JSON
{
	result: "result",
	error: "error",
}
```

A full example, using cURL:

```bash

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

#### Extended functionality (full nodes only)

| Method | Params | Description | Pre-requisites |
| ------ | ------ | ----------- | -------------- |
| `bid` | \<amount\>, \<locktime\> | Sends a bid transaction of \<amount\> DUSK to self. The transaction will be locked for \<locktime\> blocks after being accepted into a block. Returns a TXID on success. | wallet loaded |
| `stake` | \<amount\> \<locktime\> | Sends a stake transaction of \<amount\> DUSK to self. The transaction will be locked for \<locktime\> blocks after being accepted into a block. Returns a TXID on success. | wallet loaded |

