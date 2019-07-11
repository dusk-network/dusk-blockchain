

 ### General concept
For general concept explanation one can refer to /pkg/core/database/README.md. This document must focus on decisions made with regard to goleveldb specifics


### K/V storage schema to store a single `pkg/core/block.Block` into blockchain

|    Prefix   | KEY                | VALUE                    | Count           |  Used by                 |
| :-----:     | :----------------: | :---------------------:  | :----------------------:   |:----------------------:  |
|  0x01       | HeaderHash         | Header.Encode()          | 1 per block                         | 
|  0x02       | HeaderHash + TxID  | TxIndex + Tx.Encode()    | block txs count            | 
|  0x04       | TxID               | HeaderHash               | block txs count            | FetchBlockTxByHash
|  0x05       | KeyImage           | TxID                     | sum of block txs inputs    | FetchKeyImageExists
|  0x03       | Height             | HeaderHash               | 1 per block                | FetchBlockHashByHeight
|  0x07       | State              | Chain tip hash           | 1 per chain                | FetchState


### K/V storage schema to store a candidate `pkg/core/block.Block`

|    Prefix   | KEY                | VALUE                    | Count           |  Used by                 |
| :-----:     | :----------------: | :---------------------:  | :----------------------:   |:----------------------:  |
|  0x06       | HeaderHash + Height             | Block.Encode()           | Many per blockchain        | Store/Fetch/Delete CandidateBlock


Table notation
- HeaderHash - a calculated hash of block header
- TxID - a calculated hash of transaction
- \'+' operation - denotes concatenation of byte arrays
- Tx.Encode() - Encoded binary form of all Tx fields without TxID