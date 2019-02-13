Block Generation -> Block Reduction -> Binary Agreement -> Set Reduction

Outdated Notes:

		Each stage in their own folder?

			//XXX: We wait for 1 minute to receive other proposals, and take the highest score.
			// If none made from me and none received, then we move into reduction and propose the empty block
			// If one, could be from me or another person, then we use the hash of that to

			// Time to check zkproof with score, time to compare scores, time to propogate message
			// -- one minute would be enough for a small commitee

			// Where would these scores be stored?



			// Store some values in context (File)


			// Need step and round number in all messages


			// Slashing 

				- If someone is marked to be slashed, we need that slashing tx in the next block	 


-----



Kev: 

	- Refactor the GenX, GenY, GenZ... But we were in discussions to remove GenY, so maybe not yet
	- // XXX: There is a todo to fixed inputs, I believe we can only add this after we have generated a block and we can use this to also test regression. 

Jules:
	- processing.go l51 TODO
	- Can we add another test file `consensus_test` to test it from an api perspective?

	*block reduction*
		- 	// If BlockHash is nil, no clear winner was found within the time limit.
			// So we will exit and restart the consensus.
			if ctx.BlockHash == nil {
				return nil
			}
			Should we be returning a named error here? I think we should push this to the top

	- Maybe add comments to unexported methods also, to explain their functionality and what they return

	*sortition*
		- Can we move the 2^256 big int out of the function, so it's only calulated once
		- strings.Contains(err.Error(), "Invalid Signature") Can we use named errors here?
		```
		if err := ctx.BLSVerify(pk, msg, score); err != nil {
		if strings.Contains(err.Error(), "Invalid Signature") {
			return false, nil
		}

		return false, err
		}
		``` We discussed returning an error being significant, I think this can now use priority errors to simplify



