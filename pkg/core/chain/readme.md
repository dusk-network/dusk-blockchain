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


#### Notes
 

Create Issue: 	Add Certificate checks in Verify Block (After binance) 
Create Issue: 	Clear up the interface for transactions. Blocks accept merkletree.Payload, while we also have transactions.TypeInfo 
CreateIssue (Blocking): 	
In hasBlock:
```
err := c.db.View(func(tx database.Tx) error {
		_, err := tx.GetBlockHeaderByHash(blk.Header.Hash)
		return err
	})
```

This code assumes that an error is returned if a block is not found. This needs to be checked. One other possiblity, is that no error is returned and the item is nil. Better would be to implement a HasInterface, so we can do:

```
var has bool
err := c.db.View(func(tx database.Tx) error {
		has, err := tx.HasBlock(blk.Header.Hash)
		return err
	})

if err != nil {
    return err
}
if has {
    return error
}
```
CreateIssue(Blocking): We have GetBlockHeader. This should be GetBlock, as we can save headers in the database without their blocks, like in the Initial Block Download.

CreateIssue: As it stands we cannot batch the rangeproofs together to save on time, without type casting from TypeInfo. Links to tx interface issue above, ideally we do it in isBlockMalformed, while we have all of the transactions

CreateIssue: PrevBlock in the header should be prevHash
Create Issue: Units for block timestamp should be clarified (milliseconds vs seconds)

Create Issue: In checkBlockMalformed, I only check that out timestamp is more than the previous block. I should add an upper bound, whereby it cannot be > 60 minutes more. But first need to clarify whether we are in seconds or milliseconds

CreateIssue(Wire):

```

	var timestamp uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &timestamp); err != nil {
		return err
	}

	b.Timestamp = int64(timestamp)
```

why is timestamp  casted as int64 (wire)?

- Check the error wrapping, for places that I could wrap errors to make them more descriptive and helpful
- check errors are prefixed with "chain:" - wrap only api functions for this
- hasBlock returns a (bool, error) can this be shortened, so that nil means true and err means false. 

## TODO

- Modify transaction/block code in wire(blocking), add checks for transactions in chain