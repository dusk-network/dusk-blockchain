### JSON-RPC Supported API

| Method  | Param | Desc |
|---|---|---|---|
|  getLastBlock |       | Retrieve blockchain tip as json|
|  exportData |         | Export blockchain headers as json|
|  exportData |    includeBlockTxs     | Export blockchain headers and transactions as json|   
|  getMempoolTxs |      | Return current mempool state| 
|  publishEvent|        | Inject an event directly into EventBus system|


#### Configuration

```toml
[rpc]

# rpc port to listen on
port=9000
# enable rpc service
enabled=true
user="default"
pass="default"
cert=""
```

#### Example usage

```bash
# Export blockchain database as json
 curl --data-binary '{"jsonrpc":"2.0","method":"exportData","params":["includeBlockTxs"], "id":1}' -H 'content-type:application/json;' http://127.0.0.1:9000
{
        "result": "exported to /tmp/dusk-node_9000.json",
        "error": "null",
        "id": 1
}

```