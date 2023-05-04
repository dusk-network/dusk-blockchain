# Provisioners and Sortition
This package implements the data structure holding the _Provisioners_ set and the methods to extract _Block Generators_ and _Voting Committees_, which are responsible to propose and decide on new blocks during the [consensus protocol](../README.md).

## Committees
A committee is formed by a set of provisioners (called _members_), each having a certain weight, or _influence_, in the voting process. Influence is expressed as a number of vote _credits_, which are distributed among members according to a sortition process.

Each committee has a pre-defined number of available credits, which determines the degree of distribution among provisioners as well as the maximum number of members.

The influence of each member thus corresponds to the number of credits he has. When deciding on a block, each vote is counted as many times as the caster's influence in the committee. For instance, if a member has influence 3 (i.e. he has 3 credits), his vote will be counted 3 times.

> Note: to extract a block generator, a one-member committee is created by assigning a single credit to a randomly-extracted provisioner.

## Sortition
When creating a committee, available credits are assigned, one at a time, to randomly-selected provisioners. This selection is done via a sortition process called _Deterministic Sortition_, which extracts provisioners based on a cryptographic hash function. 

Credits are not distributed uniformly but favoring provisioners with higher stakes: the higher the stake, the higher the probability of being extracted. Provisioners extracted multiple times for the same committee will get more credits and hence have higher influence. 

On average, provisioners will participate in committees with a frequency and influence proportional to their stakes.

> Note: sortition is done exclusively among provisioners with _mature_ (eligible) stakes.
<!-- TODO: add a reference to _eligibility -->

### Algorithm
The _Deterministic Sortition_ algorithm assigns credits to provisioners in a pseudo-random, deterministic way. For each credit, a random provisioner is extracted, based on a cryptographic hash function. 

The hash function is applied to a concatenation of the parameters of the current consensus iteration (i.e., current block's seed, and current round and step numbers), along with the number of the credit being assigned. This ensures a unique value is generated for every single credit to assign.

## Implementation
> Note: the current implementation uses `size` to refer to the number of credits in a committee. <!-- TODO: `size` is terribly ambiguous. It should be changed to `credits` or `influence` -->

The committee creation is implemented by the `CreateVotingCommittee` function, which takes a target number of credits (`size`) and outputs the list of extracted provisioners with the corresponding influence. Committees are defined as ordered sets of couples _(provisioner, credits)_.

### Psuedocode
The following pseudocode describes the committee creation algorithm in detail.

**`CreateVotingCommittee`**

Parameters:
 - `size`: total number of voting credits in the committee
 - `W`: total stake weight of eligible provisioners
 - `seed`: previous block's seed
 - `round`, `step`: round and step number of the current consensus iteration

Procedure:
```
for i from 0 to (size-1)                       // Assign `size` credits
    1. hash = Hash(round||i||step||seed)       // Hash the concatenation of `round`, `i`, `step`, and `seed`
    2. score = Int(hash) % W                   // Set `score` `hash` (as integer) modulo `W`
    3. member = extractCommitteeMember(score)  // Select a `member` using `score`
    4. votingCommittee.addCredit(member)       // Assign a credit to `member`
    5. subtracted = (member.stake -= 1)        // Subtract up to 1 DUSK from the member's stake weight
    6. W = W - subtracted                      // Subtract the same amount from the total stake weight
    7. if W == 0: break                        // If we subtracted the whole stake weight, stop

return votingCommittee
```
---
**`extractCommitteeMember`**

Parameters:
 - `score`: pseudo-random value for current extraction
 - `Provisioners`: set of eligible provisioners (ordered by their BLS key)

Procedure:
```
i = 0                                           // Start with the first provisioner in the list
loop:
    1. provisioner = Provisioners[i]            // Get provisioner at index `i`
    2. if provisioner.stake >= score            // If the provisioner's stake is higher than `score`,
        return provisioner                      // extract the provisioner
    4. score = score - provisioner.stake        // Otherwise, decrement the score by the provisioner's stake
    5. i++ % Provisioners.size()                // And move to the next provisioner (loop over when reaching the last index)
```
