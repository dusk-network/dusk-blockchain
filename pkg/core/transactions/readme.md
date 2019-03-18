



## Notes

CreateIssue: Amount in TX cannot be a uint64, has to be bytes and masked. For now, leave as is.

CreateIssue: Need a way to have multiple commitments for one proof
    - Suggestion: for n outputs, have the commitments be present in each. For the n-1 commitment,include rangeprooof <- This needs reviewing. Verifier would aggregate each proof into one for single proofs.
    Aggregate case for prover, may leak privacy.
        For now, let's stick with one proof, one commitment 

CreateIssue: For now, in a bidding tx, the amount is in clear text. Will need to be changed to a commitment to D


CreateIssue: R cannot be used as the TxID, if a tx has more than one output, then it is ambiguous

CreateIssue: Is the sort in the Inputs slice needed, before comparing them? Contextual question

CreateIssue: We should have a limit for the number of inputs and outputs that are allowed, although
Fees can mitigate an attacker having large amounts of inputs/outputs, an attacker with a lot of money can hold up a lot of resources like this. A conservative number like 250 inputs and 100 outputs seems fair for example, or in the right ball-park for those who want to pay out employees.


CreateIssue: Specs for the coinbase tx is a bit unclear. 

CreateIssue: We can generalise equals method;

    Take in an equal interface, then swtich on the type, if types are not the same then we return false
    If they are, we carry on with our checks