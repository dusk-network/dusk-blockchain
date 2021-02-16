# Selection Phase

## Abstract

`Block Generators` \(a role taken outside of the Provisioner\) participate in a non-interactive lottery to be able to forge a candidate block. In order to decide which Block to select among the various candidates, the `Block Generators` propagate a score associated with their candidate block, which is the result of the lottery run in a non-interactive fashion. Together with the score, the `Block Generators` publish a _zero-knowledge proof_ of correctness of the score computation according to the rules outlined in the [`Blind Bid` algorithm](../blockgenerator/README.md).

