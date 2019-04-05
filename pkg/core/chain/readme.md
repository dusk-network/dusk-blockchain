## Chain

### Responsibility

- Given a block, verify whether the block is valid according to the consensus rules
- Given a transaction, verify whether a transaction is valid

### API

- AcceptBlock(block) Verifies and saves a block
- VerifyBlock(block) - Verifies a block is valid, including the transactions
- VerifyTX(tx) - Verifies a tx is valid. For the mempool

### Usage

- AcceptBlock will be used by all nodes, when they recieve a new proposed block, that should be added to the chain

- VerifyBlock will be used by consensus nodes, to verify that a block is valid without saving it.

- VerifyTX will be used by the mempool to Verify a TX is valid and can be added to the mempool.

#### Consensus Rules

- No double spending, check with utxo database
- time difference between the last block should not be more than 60 minutes
- Current blockheader, should reference the previous block header
- Seed, public key, etc in block should be X Bytes
- Zkproof should be valid in block
- Certificates in the block of provisioners, should be valid.
- Timestamp of previous block should be less than current block

#### Specification

- Chain is the only process with a RW copy to the database
