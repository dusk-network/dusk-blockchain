## Voter

### Responsibility

The voter has the responsibility to add the BLS and the ED25519 signatures and keys to the outgoing consensus messages. These outgoing messages are those published under the following topics:

    - OutgoingBlockReductionTopic
    - OutgoingBlockAgreementTopic

The Voter republishes the signed messages under the `Gossip` topic.

### API

- LaunchVotingComponent(eventbus, keys, committee) - Launches the Voter

### Architecture

Both topics have a specific `Signer`, which is a common interface exposing one method:

    - `addSignatures(wire.Event) (*bytes.Buffer, error): this function adds the correct signatures and keys to a passed event.

On launch, the `Voting Component` will spawn a `Collector` for both topics, containing a signer which is capable of handling the specific event it is poised to receive. The `Collector` receives outgoing consensus messages from the node's `EventBus`.

Thanks to the common interface, the collector code is reusable for both signers, and the only difference lies in the unmarshalling function it uses to interpret incoming data, and the specific signer logic.

The `Voting Component` also spawns a `Sender`, who is connected to the node's `EventBus`. The `Sender` publishes the resulting signed messages, that it receives from the collectors, on the `EventBus`.
