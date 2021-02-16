# Score Generator Component

## Abstract

For more information on the Blind Bid, refer to the [generation package readme](../generation.md).

## Values

### Score Event

| Field | Type |
| :--- | :--- |
| Proof | \[\]byte \(variable size\) |
| Score | \[\]byte \(32 bytes\) |
| Z | \[\]byte \(32 bytes\) |
| Bid List | \[\]byte \(variable size\) |
| Seed | BLS Signature |

## Architecture

The score generator is triggered in the generation phase of the consensus, and will be the first out of the two components to be called. It provides a sole entry point through the `Generator` interface, with the `Generate` method. It should be initialized with a DB, so that the score generator can retrieve the saved bid values before starting up.

When given a round and a step, the score generator will then construct a score generation request message, using some of the earlier retrieved bid values, and the given round and step, as well as the current consensus seed. Through a GRPC call, this request is then passed on to the RUSK server, which is responsible for calculating the actual proof.

When the proof is returned, the score generator will first check if the resulting score is high enough to be accepted by provisioners in the consensus. If so, the result is returned in the form of a `ScoreProposal` message. Otherwise, an empty message is returned.

