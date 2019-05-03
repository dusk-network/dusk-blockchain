
### Mempool
Package mempool represents the chain transaction layer (not to be confused with DB transaction layer). A blockchain transaction has one of a finite number of states at any given time. Below are listed the logical states of a transaction. 

### States

 | Name  | Desc | | Started by | Transitions  |   |
|---|---|---|---|---|---|
|  created |  tx data was built with RPC call |   | RPC subsystem| 
|  signed |  tx data was signed with RPC call |   | RPC subsystem   | 
|  received |  tx was pushed into mempool by external subsystem (RPC, P2P node, etc)| | Mempool | signed -> received   |   |
|   |  | | Mempool | propagated -> received   |   |
|  verified | tx passed the tx verification rules  | |  Mempool |  received -> verified  |   |
|  propagated | tx was gossiped to the P2P network  | |  P2P network | verified -> propagated  |   |
|  accepted | tx is part of a block that was accepted by the network  | | Mempool  | propagated -> accepted  |   |
|  stale |  tx was removed due to exceeding expiry period  | |  Mempool | any -> stale  |   |

 

Mempool participates in the following transitions
- from `created/signed` to `received`
- from `received` to `verified`
- from `verified` to `propagated`
- from `propagated` to `accepted`

### Mempool responsibilities:

- Store all txs `received` from RPC call or P2P message ready to be verified
- Store all txs `verified` by the chain ready to be included in next block
- Update itself on newly accepted block
- Monitor and report for abnormal situations

### Underlying pool

In addition, mempool tries to be storage-agnostic so that a verified tx can be stored in different forms of persistent and non-persistent pools. Supported and pending ideas for pools:

- hashmap - based on golang map implements non-persistent pool. Supported
- syncpool - based sync.Pool. Pending
- distributed - distributed memory object caching system (e.g memcached).  Pending
- persistent - persistent KV storage. Pending


### Coding stuff
Mempool implementation tries to avoid use of mutex to protect shared state. Instead, all input/output communication is based on channels. Similarily to Unix Select(..) sementics, mempool waits on read/write (input/output/timeout) channels to trigger an event to perform