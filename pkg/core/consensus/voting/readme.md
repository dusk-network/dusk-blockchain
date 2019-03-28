## Voter

### Responsibility

The voter has the responsibility to add the BLS and the ED25519 signatures to the outgoing consensus messages. These outgoing messages are those published under the following topics:

    - OutgoingBlockReductionTopic
    - OutgoingSigSetReductionTopic
    - OutgoingBlockAgreementTopic
    - OutgoingSigSetAgreementTopic

The Voter republishes the messages under the `Gossip` topic

### API

    - LaunchVotingComponent(eventbus, keys, committee) - Launches the Voter
